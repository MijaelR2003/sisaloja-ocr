import Foundation
import Vision
import AppKit

guard CommandLine.arguments.count > 1 else {
    fputs("Usage: vision-ocr <image-path>\n", stderr)
    exit(1)
}

let imagePath = CommandLine.arguments[1]
let imageURL = URL(fileURLWithPath: imagePath)

guard let nsImage = NSImage(contentsOf: imageURL),
      let cgImage = nsImage.cgImage(forProposedRect: nil, context: nil, hints: nil) else {
    fputs("Failed to load image\n", stderr)
    exit(1)
}

struct TextBox {
    let text: String
    let rect: CGRect
}

func recognize(cgImg: CGImage, orientation: CGImagePropertyOrientation) -> [TextBox] {
    var recognized: [TextBox] = []
    let request = VNRecognizeTextRequest { req, _ in
        guard let observations = req.results as? [VNRecognizedTextObservation] else { return }
        for obs in observations {
            if let top = obs.topCandidates(1).first {
                let clean = top.string.trimmingCharacters(in: .whitespacesAndNewlines)
                if !clean.isEmpty {
                    recognized.append(TextBox(text: clean, rect: obs.boundingBox))
                }
            }
        }
    }
    request.recognitionLevel = .accurate
    request.usesLanguageCorrection = false
    request.recognitionLanguages = ["es-ES", "en-US"]

    let handler = VNImageRequestHandler(cgImage: cgImg, orientation: orientation, options: [:])
    try? handler.perform([request])
    return recognized
}

let orientations: [CGImagePropertyOrientation] = [.up, .right, .left, .down]
var bestItems: [TextBox] = []
var bestScore = -1

let keywords = [
    "BOLIVIA", "PLURINACIONAL", "SEGIP", "CEDULA", "CÉDULA", "IDENTIDAD",
    "NOMBRES", "APELLIDOS", "APELLIDO", "NACIMIENTO", "IDBOL", "OCUPACION",
    "OCUPACIÓN", "PROFESION", "FIRMA", "TITULAR", "SERIE", "SECCION", "PASAPORTE"
]

for orient in orientations {
    let items = recognize(cgImg: cgImage, orientation: orient)
    let joinedUpper = items.map { $0.text.uppercased() }.joined(separator: " ")
    var score = items.count
    for kw in keywords {
        if joinedUpper.contains(kw) {
            score += 25
        }
    }
    if joinedUpper.contains("<<") {
        score += 35
    }
    if score > bestScore {
        bestScore = score
        bestItems = items
    }
    if orient == .up && score >= 30 {
        break
    }
}

// Ordenar de arriba hacia abajo (mayor midY primero)
let sortedByY = bestItems.sorted { $0.rect.midY > $1.rect.midY }

// Agrupar bloques que pertenecen a la misma línea horizontal (tolerancia Y de 1.8% de la altura)
var rows: [[TextBox]] = []
for item in sortedByY {
    if var lastRow = rows.last, let firstInRow = lastRow.first {
        if abs(item.rect.midY - firstInRow.rect.midY) <= 0.018 {
            lastRow.append(item)
            rows[rows.count - 1] = lastRow
            continue
        }
    }
    rows.append([item])
}

for row in rows {
    let sortedRow = row.sorted { $0.rect.minX < $1.rect.minX }
    let lineText = sortedRow.map { $0.text }.joined(separator: "   ")
    print(lineText)
}
