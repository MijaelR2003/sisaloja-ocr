package ocr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/MijaelR2003/sisaloja-ocr/internal/parser"
)

// findVisionBinary localiza el binario nativo vision-ocr (Apple Vision Neural Engine en macOS).
func findVisionBinary() string {
	candidates := []string{"./vision-ocr"}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "vision-ocr"))
	}
	if p, err := exec.LookPath("vision-ocr"); err == nil {
		candidates = append(candidates, p)
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

// HasVisionOCR indica si el motor neuronal Apple Vision está disponible en el servidor.
func HasVisionOCR() bool {
	return findVisionBinary() != ""
}

// ExtractTextWithVisionOCR ejecuta el motor neuronal Apple Vision sobre bytes de imagen.
func ExtractTextWithVisionOCR(imgBytes []byte) (string, error) {
	if len(imgBytes) == 0 {
		return "", nil
	}
	bin := findVisionBinary()
	if bin == "" {
		return "", errors.New("vision-ocr no disponible")
	}
	tmpFile, err := os.CreateTemp("", "sisaloja-vision-*.jpg")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(imgBytes); err != nil {
		tmpFile.Close()
		return "", err
	}
	tmpFile.Close()

	cmd := exec.Command(bin, tmpPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// HasTesseractCLI comprueba si el binario tesseract está disponible en el sistema operativo o contenedor.
func HasTesseractCLI() bool {
	_, err := exec.LookPath("tesseract")
	return err == nil
}

// ExtractTextWithTesseractCLI ejecuta tesseract sobre bytes de imagen si el binario está disponible.
func ExtractTextWithTesseractCLI(imgBytes []byte, lang string) (string, error) {
	if len(imgBytes) == 0 {
		return "", nil
	}
	bin, err := exec.LookPath("tesseract")
	if err != nil {
		return "", errors.New("tesseract CLI no disponible en este host")
	}
	if lang == "" {
		lang = "spa"
	}

	tmpFile, err := os.CreateTemp("", "sisaloja-ocr-*.png")
	if err != nil {
		return "", err
	}
	tmpPath := tmpFile.Name()
	defer os.Remove(tmpPath)

	if _, err := tmpFile.Write(imgBytes); err != nil {
		tmpFile.Close()
		return "", err
	}
	tmpFile.Close()

	cmd := exec.Command(bin, tmpPath, "stdout", "-l", lang)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		// Reintentar con idioma por defecto si 'spa' no está instalado
		cmdFallback := exec.Command(bin, tmpPath, "stdout")
		out.Reset()
		cmdFallback.Stdout = &out
		if err2 := cmdFallback.Run(); err2 != nil {
			return "", fmt.Errorf("error ejecutando tesseract: %w", err)
		}
	}
	return out.String(), nil
}

// DecodeBase64Image extrae los bytes de una cadena Base64 o DataURL (data:image/jpeg;base64,...).
func DecodeBase64Image(dataURL string) ([]byte, error) {
	clean := strings.TrimSpace(dataURL)
	if clean == "" {
		return nil, nil
	}
	if idx := strings.Index(clean, ","); idx != -1 && strings.HasPrefix(clean[:idx], "data:") {
		clean = clean[idx+1:]
	}
	return base64.StdEncoding.DecodeString(clean)
}

// ExtractWithGeminiVision usa Gemini 2.5 Flash si GEMINI_API_KEY está configurada en el entorno.
func ExtractWithGeminiVision(apiKey string, frontImg, backImg []byte) (*parser.DocumentFields, error) {
	if strings.TrimSpace(apiKey) == "" || (len(frontImg) == 0 && len(backImg) == 0) {
		return nil, errors.New("gemini vision no configurado")
	}

	var parts []map[string]interface{}
	parts = append(parts, map[string]interface{}{
		"text": `Extrae los datos de este documento de identidad (Cédula de Bolivia, DNI o Pasaporte) y responde ÚNICAMENTE con un objeto JSON válido con las claves exactas: document_type ("CI"|"DNI"|"PASAPORTE"), document_number, full_name, nationality, gender ("M"|"F"), birth_date ("YYYY-MM-DD"), occupation, origin_city.`,
	})
	for _, img := range [][]byte{frontImg, backImg} {
		if len(img) > 0 {
			parts = append(parts, map[string]interface{}{
				"inline_data": map[string]interface{}{
					"mime_type": "image/jpeg",
					"data":      base64.StdEncoding.EncodeToString(img),
				},
			})
		}
	}

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": parts},
		},
		"generationConfig": map[string]interface{}{
			"responseMimeType": "application/json",
		},
	}
	bodyBytes, _ := json.Marshal(payload)

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s", filepath.Clean(apiKey))
	client := &http.Client{Timeout: 12 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini api status %d", resp.StatusCode)
	}

	respBytes, _ := io.ReadAll(resp.Body)
	var gemResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(respBytes, &gemResp); err != nil || len(gemResp.Candidates) == 0 || len(gemResp.Candidates[0].Content.Parts) == 0 {
		return nil, errors.New("respuesta inválida de gemini")
	}

	var fields parser.DocumentFields
	if err := json.Unmarshal([]byte(gemResp.Candidates[0].Content.Parts[0].Text), &fields); err != nil {
		return nil, err
	}
	return &fields, nil
}
