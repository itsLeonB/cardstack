import {
  existsSync,
  mkdirSync,
  mkdtempSync,
  readdirSync,
  rmSync,
  writeFileSync,
} from "node:fs"
import { tmpdir } from "node:os"
import { join } from "node:path"
import { decode, encode } from "jpeg-js"
import { afterAll, beforeAll, describe, expect, it } from "vitest"
import { cellRegions, cellSize } from "../src/lib/page-cut"
import type { Corners } from "../src/lib/page-cut"
import {
  applyOrientation,
  cutFolder,
  cutPhoto,
  downscale,
  exifOrientation,
  parseGrid,
  readCorners,
  runCli,
} from "./cut-pages"
import type { RgbaImage } from "./cut-pages"

const whole: Corners = [
  { x: 0, y: 0 },
  { x: 1, y: 0 },
  { x: 1, y: 1 },
  { x: 0, y: 1 },
]

// Left half black, right half white, so a cell's colour says which half it was cut from.
function halvesJpeg(width: number, height: number): Uint8Array {
  const data = new Uint8Array(width * height * 4)
  for (let at = 0; at < data.length; at += 4) {
    const x = (at / 4) % width
    data.set([x < width / 2 ? 0 : 255, 0, 0, 255], at)
  }
  return encode({ data, width, height }, 90).data
}

// A JPEG whose APP1 segment carries only an EXIF Orientation tag.
function exifJpeg(orientation: number): Uint8Array {
  const tiff = [
    0x49,
    0x49,
    0x2a,
    0x00,
    8,
    0,
    0,
    0, // little-endian TIFF header, IFD at byte 8
    1,
    0, // one entry
    0x12,
    0x01,
    3,
    0,
    1,
    0,
    0,
    0,
    orientation,
    0,
    0,
    0, // Orientation, SHORT, count 1
    0,
    0,
    0,
    0, // no next IFD
  ]
  const app1 = [0x45, 0x78, 0x69, 0x66, 0, 0, ...tiff] // "Exif\0\0" + TIFF
  const length = app1.length + 2
  return Uint8Array.from([
    0xff,
    0xd8,
    0xff,
    0xe1,
    length >> 8,
    length & 0xff,
    ...app1,
    0xff,
    0xd9,
  ])
}

// A 2-wide, 3-tall image whose pixel at (x, y) holds x in red and y in green.
function coordinateImage(): RgbaImage {
  const width = 2
  const height = 3
  const pixels = new Uint8ClampedArray(width * height * 4)
  for (let y = 0; y < height; y++)
    for (let x = 0; x < width; x++)
      pixels.set([x, y, 0, 255], (y * width + x) * 4)
  return { width, height, pixels }
}

function pixelAt(image: RgbaImage, x: number, y: number) {
  const i = (y * image.width + x) * 4
  return [image.pixels[i], image.pixels[i + 1]]
}

describe("parseGrid", () => {
  it("reads rows x columns", () => {
    expect(parseGrid("3x4")).toEqual({ rows: 3, columns: 4 })
  })

  it.each([undefined, "", "3", "3x", "x3", "0x3", "3x0", "3*3", "axb"])(
    "rejects %s with the usage hint",
    (text) => {
      expect(() => parseGrid(text)).toThrow("--grid must be rows x columns")
    }
  )
})

describe("readCorners", () => {
  it("reads four [x, y] pairs in order", () => {
    const corners = readCorners(
      "[[0.1,0.1],[0.9,0.1],[0.9,0.9],[0.1,0.9]]",
      "page.corners.json"
    )
    expect(corners[2]).toEqual({ x: 0.9, y: 0.9 })
  })

  it("rejects text that is not JSON", () => {
    expect(() => readCorners("[[0,0]", "page.corners.json")).toThrow(
      "page.corners.json is not valid JSON"
    )
  })

  it("rejects a shape other than four pairs", () => {
    expect(() =>
      readCorners("[[0,0],[1,0],[1,1]]", "page.corners.json")
    ).toThrow("page.corners.json must be [[x,y],[x,y],[x,y],[x,y]]")
  })

  it("rejects corners the app would reject", () => {
    expect(() =>
      readCorners("[[0,0],[1,1],[1,0],[0,1]]", "page.corners.json")
    ).toThrow(/^page\.corners\.json: Move the corners/)
    expect(() =>
      readCorners("[[0,0],[1.5,0],[1,1],[0,1]]", "page.corners.json")
    ).toThrow(/^page\.corners\.json: The corners must stay on the photo/)
  })
})

describe("exifOrientation", () => {
  it("reads the Orientation tag", () => {
    expect(exifOrientation(exifJpeg(6))).toBe(6)
  })

  it("is 1 when the JPEG has no EXIF", () => {
    expect(exifOrientation(halvesJpeg(4, 4))).toBe(1)
  })
})

describe("applyOrientation", () => {
  it("turns a portrait-stored photo upright for orientation 6 (rotate clockwise)", () => {
    const out = applyOrientation(coordinateImage(), 6)

    expect(out.width).toBe(3)
    expect(out.height).toBe(2)
    expect(pixelAt(out, 2, 0)).toEqual([0, 0]) // stored top-left is now top-right
    expect(pixelAt(out, 0, 1)).toEqual([1, 2]) // stored bottom-right is now bottom-left
  })

  it("turns the photo the other way for orientation 8 (rotate counter-clockwise)", () => {
    const out = applyOrientation(coordinateImage(), 8)

    expect(pixelAt(out, 0, 1)).toEqual([0, 0]) // stored top-left is now bottom-left
    expect(pixelAt(out, 2, 0)).toEqual([1, 2]) // stored bottom-right is now top-right
  })

  it("leaves orientation 1 alone", () => {
    expect(applyOrientation(coordinateImage(), 1)).toEqual(coordinateImage())
  })
})

describe("downscale", () => {
  it("averages each block of source pixels into one output pixel", () => {
    const image: RgbaImage = {
      width: 4,
      height: 2,
      pixels: new Uint8ClampedArray(
        [0, 2, 4, 6, 0, 2, 4, 6].flatMap((r) => [r, 0, 0, 255])
      ),
    }

    const out = downscale(image, 2, 1)

    expect(out.width).toBe(2)
    expect(out.height).toBe(1)
    expect(out.pixels[0]).toBe(1)
    expect(out.pixels[4]).toBe(5)
  })
})

describe("cutPhoto", () => {
  it("cuts a synthetic JPEG into one JPEG per cell, sized as the app sizes them", () => {
    const grid = { rows: 1, columns: 2 }

    const cells = cutPhoto(halvesJpeg(40, 20), whole, grid)

    expect(cells.map(({ row, column }) => [row, column])).toEqual([
      [1, 1],
      [1, 2],
    ])
    const [left, right] = cells.map((cell) =>
      decode(cell.jpeg, { useTArray: true })
    )
    const expected = cellSize(whole, grid, cellRegions(grid)[0], 40, 20)
    expect(left.width).toBe(expected.w)
    expect(left.height).toBe(expected.h)
    expect(left.data[0]).toBeLessThan(40) // black half
    expect(right.data[0]).toBeGreaterThan(215) // white half
  })
})

describe("cutFolder", () => {
  let root: string

  beforeAll(() => {
    root = mkdtempSync(join(tmpdir(), "cut-pages-"))
  })

  afterAll(() => {
    rmSync(root, { recursive: true, force: true })
  })

  // Each test gets its own photos folder and output folder.
  function fixture(name: string, files: Record<string, Uint8Array | string>) {
    const photos = join(root, name, "photos")
    mkdirSync(photos, { recursive: true })
    for (const [file, content] of Object.entries(files)) {
      writeFileSync(join(photos, file), content)
    }
    return { photos, out: join(root, name, "out") }
  }

  it("cuts every photo in name order into r<RR>c<CC>.jpg folders and prints one line each", () => {
    const { photos, out } = fixture("names", {
      "b.jpg": halvesJpeg(40, 20),
      "a.jpeg": halvesJpeg(40, 20),
      "notes.txt": "not a photo",
    })

    const lines = cutFolder(photos, out, { rows: 1, columns: 2 })

    expect(lines).toHaveLength(2)
    expect(lines[0]).toContain("a.jpeg")
    expect(lines[0]).toContain("whole photo")
    expect(readdirSync(join(out, "a")).sort()).toEqual([
      "r01c01.jpg",
      "r01c02.jpg",
    ])
    expect(readdirSync(join(out, "b")).sort()).toEqual([
      "r01c01.jpg",
      "r01c02.jpg",
    ])
  })

  it("reads the corners sidecar next to a photo", () => {
    const trapezoid = "[[0.2,0.1],[0.8,0.15],[0.9,0.9],[0.1,0.85]]"
    const { photos, out } = fixture("sidecar", {
      "page.jpg": halvesJpeg(40, 20),
      "page.corners.json": trapezoid,
    })

    const lines = cutFolder(photos, out, { rows: 1, columns: 2 })

    expect(lines[0]).toContain("page.corners.json")
    expect(readdirSync(join(out, "page")).sort()).toHaveLength(2)
  })

  it("rejects a bad sidecar before writing any output", () => {
    const { photos, out } = fixture("bad-sidecar", {
      "page.jpg": halvesJpeg(40, 20),
      "page.corners.json": "[[0,0],[1,1],[1,0],[0,1]]",
    })

    expect(() => cutFolder(photos, out, { rows: 1, columns: 2 })).toThrow(
      /^page\.corners\.json: Move the corners/
    )
    expect(existsSync(out)).toBe(false)
  })

  it("fails clearly on a missing photos folder", () => {
    expect(() =>
      cutFolder(join(root, "nowhere"), join(root, "nowhere-out"), {
        rows: 1,
        columns: 1,
      })
    ).toThrow("photos folder not found")
  })

  it("fails clearly when the folder has no photos", () => {
    const { photos, out } = fixture("empty", { "notes.txt": "hello" })

    expect(() => cutFolder(photos, out, { rows: 1, columns: 1 })).toThrow(
      "no .jpg or .jpeg photos in"
    )
  })
})

describe("runCli", () => {
  it("needs --grid and two folders", () => {
    expect(() => runCli(["photos", "out"])).toThrow(
      "usage: bun tools/cut-pages.ts <photosDir> <outDir> --grid <rows>x<columns>"
    )
    expect(() => runCli(["photos", "--grid", "3x3", "out", "extra"])).toThrow(
      "usage:"
    )
  })

  it("rejects a bad grid before reading the folders", () => {
    expect(() => runCli(["photos", "out", "--grid", "3"])).toThrow(
      "--grid must be rows x columns"
    )
  })
})
