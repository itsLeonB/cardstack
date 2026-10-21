import {
  existsSync,
  mkdirSync,
  readdirSync,
  readFileSync,
  statSync,
  writeFileSync,
} from "node:fs"
import { join } from "node:path"
import { decode, encode } from "jpeg-js"
import { z } from "zod"
import { fitLongSide } from "../src/lib/frame-crop"
import {
  cellRegions,
  cornersProblem,
  cutCellPixels,
  frameLongSide,
} from "../src/lib/page-cut"
import type { Corners, Grid } from "../src/lib/page-cut"

// Offline cutter for the tuning script. It cuts each Page photo with the app's own `cutCellPixels`,
// after the same downscale `ScanPageFlow` applies to a picked file, so the numbers match the app.
//
//   bun tools/cut-pages.ts <photosDir> <outDir> --grid <rows>x<columns>
//
// Writes <outDir>/<photo-basename>/r<RR>c<CC>.jpg. Corners come from <photo-basename>.corners.json
// beside the photo ([[x,y],...] in top-left, top-right, bottom-right, bottom-left order, 0 to 1);
// without one the whole photo is the page. jpeg-js ignores EXIF, so the orientation is applied here,
// as the browser does when it decodes a picked file.

export interface RgbaImage {
  width: number
  height: number
  pixels: Uint8ClampedArray
}

const CELL_JPEG_QUALITY = 90
const WHOLE_PHOTO: Corners = [
  { x: 0, y: 0 },
  { x: 1, y: 0 },
  { x: 1, y: 1 },
  { x: 0, y: 1 },
]
const USAGE =
  "usage: bun tools/cut-pages.ts <photosDir> <outDir> --grid <rows>x<columns>"
const POINT = z.tuple([z.number(), z.number()])
const CORNERS_JSON = z.tuple([POINT, POINT, POINT, POINT])

export function parseGrid(text: string | undefined): Grid {
  const match = /^(\d+)x(\d+)$/.exec(text ?? "")
  if (!match || Number(match[1]) < 1 || Number(match[2]) < 1)
    throw new Error("--grid must be rows x columns, like 3x3")
  return { rows: Number(match[1]), columns: Number(match[2]) }
}

export function readCorners(text: string, label: string): Corners {
  let json: unknown
  try {
    json = JSON.parse(text)
  } catch {
    throw new Error(`${label} is not valid JSON`)
  }
  const parsed = CORNERS_JSON.safeParse(json)
  if (!parsed.success)
    throw new Error(`${label} must be [[x,y],[x,y],[x,y],[x,y]]`)
  const [a, b, c, d] = parsed.data
  const corners: Corners = [
    { x: a[0], y: a[1] },
    { x: b[0], y: b[1] },
    { x: c[0], y: c[1] },
    { x: d[0], y: d[1] },
  ]
  const problem = cornersProblem(corners)
  if (problem) throw new Error(`${label}: ${problem}`)
  return corners
}

// The EXIF orientation (1 to 8) the browser applies to a JPEG on decode, or 1 without one.
export function exifOrientation(bytes: Uint8Array): number {
  const view = new DataView(bytes.buffer, bytes.byteOffset, bytes.byteLength)
  if (bytes.length < 4 || view.getUint16(0) !== 0xffd8) return 1
  let at = 2
  while (at + 4 <= bytes.length && view.getUint8(at) === 0xff) {
    const marker = view.getUint8(at + 1)
    if (marker === 0xda) break // start of scan: the headers are over
    const length = view.getUint16(at + 2)
    const isExif =
      marker === 0xe1 &&
      at + 10 <= bytes.length &&
      view.getUint32(at + 4) === 0x45786966 // "Exif"
    if (isExif) return readOrientation(view, at + 10)
    at += 2 + length
  }
  return 1
}

function readOrientation(view: DataView, tiff: number) {
  const little = view.getUint16(tiff) === 0x4949 // "II"
  const ifd = tiff + view.getUint32(tiff + 4, little)
  const count = view.getUint16(ifd, little)
  for (let i = 0; i < count; i++) {
    const entry = ifd + 2 + i * 12
    if (view.getUint16(entry, little) === 0x0112)
      return view.getUint16(entry + 8, little)
  }
  return 1
}

// Where a stored pixel (x, y) of a w x h image lands once the orientation is applied, per the EXIF spec.
const TO_DISPLAY: Array<
  (x: number, y: number, w: number, h: number) => [number, number]
> = [
  (x, y) => [x, y],
  (x, y, w) => [w - 1 - x, y],
  (x, y, w, h) => [w - 1 - x, h - 1 - y],
  (x, y, _w, h) => [x, h - 1 - y],
  (x, y) => [y, x],
  (x, y, _w, h) => [h - 1 - y, x],
  (x, y, w, h) => [h - 1 - y, w - 1 - x],
  (x, y, w) => [y, w - 1 - x],
]

export function applyOrientation(
  image: RgbaImage,
  orientation: number
): RgbaImage {
  if (orientation === 1) return image
  const { width: w, height: h, pixels } = image
  const turned = orientation >= 5
  const width = turned ? h : w
  const height = turned ? w : h
  const out = new Uint8ClampedArray(pixels.length)
  const place = TO_DISPLAY[orientation - 1]
  for (let y = 0; y < h; y++) {
    for (let x = 0; x < w; x++) {
      const [dx, dy] = place(x, y, w, h)
      const from = (y * w + x) * 4
      const to = (dy * width + dx) * 4
      out.set(pixels.subarray(from, from + 4), to)
    }
  }
  return { width, height, pixels: out }
}

// Averages each block of source pixels into one output pixel: a box filter, the offline stand-in for the canvas downscale.
export function downscale(
  image: RgbaImage,
  width: number,
  height: number
): RgbaImage {
  const out = new Uint8ClampedArray(width * height * 4)
  const sx = image.width / width
  const sy = image.height / height
  for (let j = 0; j < height; j++) {
    const y0 = Math.floor(j * sy)
    const y1 = Math.max(y0 + 1, Math.floor((j + 1) * sy))
    for (let i = 0; i < width; i++) {
      const x0 = Math.floor(i * sx)
      const x1 = Math.max(x0 + 1, Math.floor((i + 1) * sx))
      const sum = [0, 0, 0, 0]
      let count = 0
      for (let y = y0; y < y1; y++) {
        for (let x = x0; x < x1; x++) {
          const at = (y * image.width + x) * 4
          for (let c = 0; c < 4; c++) sum[c] += image.pixels[at + c]
          count++
        }
      }
      out.set(
        sum.map((v) => v / count),
        (j * width + i) * 4
      )
    }
  }
  return { width, height, pixels: out }
}

function decodeRgba(bytes: Uint8Array): RgbaImage {
  const { width, height, data } = decode(bytes, {
    useTArray: true,
    formatAsRGBA: true,
  })
  return {
    width,
    height,
    pixels: new Uint8ClampedArray(
      data.buffer,
      data.byteOffset,
      data.byteLength
    ),
  }
}

function encodeJpeg(image: RgbaImage): Uint8Array {
  return encode(
    { data: image.pixels, width: image.width, height: image.height },
    CELL_JPEG_QUALITY
  ).data
}

export function cutPhoto(
  bytes: Uint8Array,
  corners: Corners,
  grid: Grid
): Array<{ row: number; column: number; jpeg: Uint8Array }> {
  const photo = applyOrientation(decodeRgba(bytes), exifOrientation(bytes))
  const size = fitLongSide(photo.width, photo.height, frameLongSide(grid))
  // The app cuts a JPEG of the downscaled frame, so the cut sees the same encoding round trip.
  const frame = decodeRgba(encodeJpeg(downscale(photo, size.w, size.h)))
  return cellRegions(grid).map((cell) => {
    const cut = cutCellPixels(
      frame.pixels,
      frame.width,
      frame.height,
      corners,
      grid,
      cell
    )
    return { row: cell.row, column: cell.column, jpeg: encodeJpeg(cut) }
  })
}

const isJpegName = (name: string) => /\.jpe?g$/i.test(name)
const photoBase = (name: string) => name.replace(/\.jpe?g$/i, "")
const pad = (n: number) => String(n).padStart(2, "0")

export function cutFolder(
  photosDir: string,
  outDir: string,
  grid: Grid
): string[] {
  if (!existsSync(photosDir) || !statSync(photosDir).isDirectory())
    throw new Error(`photos folder not found: ${photosDir}`)
  const photos = readdirSync(photosDir).filter(isJpegName).sort()
  if (photos.length === 0)
    throw new Error(`no .jpg or .jpeg photos in ${photosDir}`)

  // Every sidecar is read before the first write, so one bad sidecar leaves no partial output.
  const jobs = photos.map((file) => {
    const sidecar = `${photoBase(file)}.corners.json`
    const sidecarPath = join(photosDir, sidecar)
    const hasSidecar = existsSync(sidecarPath)
    const corners = hasSidecar
      ? readCorners(readFileSync(sidecarPath, "utf8"), sidecar)
      : WHOLE_PHOTO
    return { file, corners, source: hasSidecar ? sidecar : "whole photo" }
  })

  return jobs.map(({ file, corners, source }) => {
    let cells: ReturnType<typeof cutPhoto>
    try {
      cells = cutPhoto(readFileSync(join(photosDir, file)), corners, grid)
    } catch (error) {
      const reason = error instanceof Error ? error.message : String(error)
      throw new Error(`${file}: could not cut (${reason})`)
    }
    const dir = join(outDir, photoBase(file))
    mkdirSync(dir, { recursive: true })
    for (const cell of cells) {
      writeFileSync(
        join(dir, `r${pad(cell.row)}c${pad(cell.column)}.jpg`),
        cell.jpeg
      )
    }
    return `${file} -> ${dir} (${grid.rows}x${grid.columns} cells, ${source})`
  })
}

export function runCli(argv: string[]): string[] {
  const at = argv.indexOf("--grid")
  if (at < 0) throw new Error(USAGE)
  const folders = [...argv.slice(0, at), ...argv.slice(at + 2)]
  if (folders.length !== 2) throw new Error(USAGE)
  return cutFolder(folders[0], folders[1], parseGrid(argv[at + 1]))
}

if (process.argv[1]?.endsWith("cut-pages.ts")) {
  try {
    for (const line of runCli(process.argv.slice(2))) console.log(line)
  } catch (error) {
    const reason = error instanceof Error ? error.message : String(error)
    console.error(`[cut-pages] ${reason}`)
    process.exit(1)
  }
}
