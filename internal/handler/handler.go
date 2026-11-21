package handler

import (
	"io"
	"os"
	"strings"

	"github.com/MijaelR2003/sisaloja-ocr/internal/ocr"
	"github.com/MijaelR2003/sisaloja-ocr/internal/parser"
	"github.com/gofiber/fiber/v2"
)

type OCRHandler struct {
	TesseractLang string
	GeminiAPIKey  string
}

func NewOCRHandler() *OCRHandler {
	lang := os.Getenv("TESSERACT_LANG")
	if lang == "" {
		lang = "spa"
	}
	return &OCRHandler{
		TesseractLang: lang,
		GeminiAPIKey:  strings.TrimSpace(os.Getenv("GEMINI_API_KEY")),
	}
}

type scanRequestJSON struct {
	FrontText   string `json:"front_text"`
	BackText    string `json:"back_text"`
	FrontBase64 string `json:"front_image_base64"`
	BackBase64  string `json:"back_image_base64"`
}

// Health: GET /health
func (h *OCRHandler) Health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":  "ok",
		"service": "sisaloja-ocr",
		"port":    8090,
		"engines": fiber.Map{
			"mrz_icao9303":        true,
			"segip_bolivia":       true,
			"apple_vision_neural": ocr.HasVisionOCR(),
			"tesseract_cli":       ocr.HasTesseractCLI(),
			"gemini_vision":       h.GeminiAPIKey != "",
		},
	})
}

// ParseText: POST /api/v1/ocr/parse-text
func (h *OCRHandler) ParseText(c *fiber.Ctx) error {
	var req scanRequestJSON
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "Formato JSON inválido",
			},
		})
	}
	if strings.TrimSpace(req.FrontText) == "" && strings.TrimSpace(req.BackText) == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "VALIDATION_ERROR",
				"message": "Debe enviar front_text o back_text para analizar",
			},
		})
	}

	res := parser.ParseAndMergeSides(req.FrontText, req.BackText)
	return c.Status(fiber.StatusOK).JSON(res)
}

// ScanDocument: POST /api/v1/ocr/scan
// Soporta tanto JSON (front_text/back_text/front_image_base64/back_image_base64)
// como multipart/form-data (front_image, back_image, front_text, back_text).
func (h *OCRHandler) ScanDocument(c *fiber.Ctx) error {
	var frontText, backText string
	var frontBytes, backBytes []byte

	contentType := strings.ToLower(c.Get("Content-Type"))
	if strings.HasPrefix(contentType, "multipart/form-data") {
		frontText = c.FormValue("front_text")
		backText = c.FormValue("back_text")

		if fh, err := c.FormFile("front_image"); err == nil && fh != nil {
			if f, errOpen := fh.Open(); errOpen == nil {
				frontBytes, _ = io.ReadAll(f)
				f.Close()
			}
		}
		if bh, err := c.FormFile("back_image"); err == nil && bh != nil {
			if f, errOpen := bh.Open(); errOpen == nil {
				backBytes, _ = io.ReadAll(f)
				f.Close()
			}
		}
	} else {
		var req scanRequestJSON
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": fiber.Map{
					"code":    "VALIDATION_ERROR",
					"message": "Cuerpo de petición inválido",
				},
			})
		}
		frontText = req.FrontText
		backText = req.BackText
		if req.FrontBase64 != "" {
			frontBytes, _ = ocr.DecodeBase64Image(req.FrontBase64)
		}
		if req.BackBase64 != "" {
			backBytes, _ = ocr.DecodeBase64Image(req.BackBase64)
		}
	}

	// 1. Si llegaron bytes de imagen y el servidor tiene Apple Vision Neural OCR (macOS) o Tesseract CLI,
	// extraer el texto de alta precisión directamente en el servidor.
	var visionFront, visionBack string
	usedVision := false
	if len(frontBytes) > 0 && ocr.HasVisionOCR() {
		if txt, err := ocr.ExtractTextWithVisionOCR(frontBytes); err == nil && strings.TrimSpace(txt) != "" {
			visionFront = txt
			usedVision = true
		}
	} else if len(frontBytes) > 0 && ocr.HasTesseractCLI() {
		if txt, err := ocr.ExtractTextWithTesseractCLI(frontBytes, h.TesseractLang); err == nil && strings.TrimSpace(txt) != "" {
			frontText = txt + "\n" + frontText
		}
	}

	if len(backBytes) > 0 && ocr.HasVisionOCR() {
		if txt, err := ocr.ExtractTextWithVisionOCR(backBytes); err == nil && strings.TrimSpace(txt) != "" {
			visionBack = txt
			usedVision = true
		}
	} else if len(backBytes) > 0 && ocr.HasTesseractCLI() {
		if txt, err := ocr.ExtractTextWithTesseractCLI(backBytes, h.TesseractLang); err == nil && strings.TrimSpace(txt) != "" {
			backText = txt + "\n" + backText
		}
	}

	// 2. Ejecutar el decodificador MRZ ICAO 9303 + Motor Especializado SEGIP Bolivia
	var res parser.ParseResult
	if usedVision {
		// Analizar primero el texto limpio del motor neuronal Apple Vision sin ruido de Tesseract
		res = parser.ParseAndMergeSides(visionFront, visionBack)
		res.Engine = "apple-vision-neural+segip-bolivia"
		res.RawText = strings.TrimSpace(visionFront + "\n" + visionBack)

		// Si el navegador también envió texto adicional (ej. de un recorte ROI), completar solo campos que falten
		if strings.TrimSpace(frontText) != "" || strings.TrimSpace(backText) != "" {
			fallbackRes := parser.ParseAndMergeSides(frontText, backText)
			res.Fields = parser.MergeFieldsWithMRZPriority(res.Fields, res.DetectedMRZ, fallbackRes.Fields, fallbackRes.DetectedMRZ)
			if fallbackRes.DetectedMRZ {
				res.DetectedMRZ = true
			}
		}
	} else {
		res = parser.ParseAndMergeSides(frontText, backText)
		res.RawText = strings.TrimSpace(frontText + "\n" + backText)
	}

	// 3. Si hay GEMINI_API_KEY opcional configurada y llegaron imágenes, enriquecer campos vacíos
	if h.GeminiAPIKey != "" && (len(frontBytes) > 0 || len(backBytes) > 0) {
		if gemFields, err := ocr.ExtractWithGeminiVision(h.GeminiAPIKey, frontBytes, backBytes); err == nil && gemFields != nil {
			res.Fields = parser.MergeFields(res.Fields, *gemFields)
			res.Engine = "gemini-vision+mrz"
			res.Confidence = "ALTA (IA Multimodal)"
		}
	}

	return c.Status(fiber.StatusOK).JSON(res)
}
