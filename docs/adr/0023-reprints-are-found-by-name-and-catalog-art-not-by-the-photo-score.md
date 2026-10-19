# Reprints are found by name and catalog art, on demand, not by the photo's score

One artwork is printed in several Expansion Sets with the same name and nearly identical pictures, so the matcher's top 5 can fill with one Card's prints and miss the print the user holds. The matcher cannot separate them by the picture, because the picture is the same; the printed set is the part the user has to choose. So the API also offers a Card's Reprints (GLOSSARY.md).

A Reprint is found from the Card ids the client holds, not from the photo. For each of up to five ids, the API takes Cards with the same normalized name (case and whitespace folded, otherwise exact, so "Pikachu ex" is not "Pikachu"), excludes every id the client sent, and keeps those whose catalog embedding lies within a cosine delta (`MATCH_REPRINT_DELTA`, starting at 0.05) of that source Card's catalog embedding. That distance compares two catalog images with each other, so it needs neither the photo nor its vector, and it separates the same art from a different art under the same name. Results are ordered by source rank, then by closeness to the source, then newest Expansion Set first; a Card reachable from two sources appears once, at its first place. Cards with no embedding and Cards in unpublished Expansion Sets are left out. `GET /scan/reprints?cardIds=<id>&cardIds=<id>&offset=<n>` returns `{cards, hasMore}` in pages of five, and the same ids give the same order on every call, so an offset is stable without any server state.

Reprints are fetched only when the user opens "Other prints", never in the match response, and the entry is offered on every result, confident ones included: a delta wider than the confident margin means a confident winner can still have a Reprint, and a confident wrong print is the costly mistake because no picker is shown. A confident result sends the one candidate id it has. The list runs until the Reprints are exhausted; it does not continue into nearest neighbours of the photo.

## Considered Options

- Continue the stream past the Reprints into plain nearest neighbours by score, ending only when the catalog is exhausted. Rejected by the operator: after the Reprints are gone, the user can mark the Scan Unidentified.
- Return a vector in the match response and send it back with a cursor, so the order follows the photo's score. Rejected in favour of ids: nothing leaves the client but ids, the API stays stateless and no extra embedding call is made.
- Name alone, without the art test. Rejected: hundreds of Pikachu with different art share a name, and Basic Energy shares one across dozens of Expansion Sets.
- Include the first page of Reprints in the match response. Rejected: the top 5 already are the alternatives the user sees, and the lookup is only needed as a fallback.

## Consequences

The art test rests on catalog embeddings, so a Card without a hosted image cannot be offered as a Reprint, matching the existing rule that it cannot be matched. The delta of 0.05 is a first guess: too tight hides real Reprints, too loose floods the list with other art, and it only changes the contents and order of this list, so it is cheap to retune against same-name groups in the catalog. Two distinct Cards that share a name and art but differ in a way that matters to a collector (a changed frame or a regulation mark) are offered as Reprints and the user decides. When a Card gains several embeddings (a later change), the art test takes the nearest one.
