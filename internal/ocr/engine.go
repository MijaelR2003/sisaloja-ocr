package ocr

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"math"
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

// ExtractTextWithTesseractCLI ejecuta tesseract sobre bytes de imagen pre-procesados en escala de grises
// con normalización local de iluminación y una segunda pasada específica para la zona MRZ (IDBOL<<<).
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

	decoded, _, decErr := image.Decode(bytes.NewReader(imgBytes))
	if decErr != nil {
		// Formato no decodificado por stdlib (ej. WebP): intentar pasar bytes crudos directo a Leptonica/Tesseract
		tmpRaw, errTmp := os.CreateTemp("", "sisaloja-raw-*")
		if errTmp != nil {
			return "", errTmp
		}
		rawPath := tmpRaw.Name()
		defer os.Remove(rawPath)
		if _, errW := tmpRaw.Write(imgBytes); errW != nil {
			tmpRaw.Close()
			return "", errW
		}
		tmpRaw.Close()
		return runTesseractArgs(bin, rawPath, lang, "--oem", "1", "--psm", "6", "--dpi", "300", "-c", "preserve_interword_spaces=1")
	}

	// 1. Escalar a mínimo 1600px de lado mayor para que etiquetas pequeñas (NOMBRES/APELLIDOS) sean legibles
	b := decoded.Bounds()
	w, h := b.Dx(), b.Dy()
	maxSide := w
	if h > maxSide {
		maxSide = h
	}
	targetImg := decoded
	if maxSide > 0 && maxSide < 1600 {
		scale := 1600.0 / float64(maxSide)
		if scale > 2.5 {
			scale = 2.5
		}
		targetImg = resizeBilinear(decoded, int(math.Round(float64(w)*scale)), int(math.Round(float64(h)*scale)))
	}

	normGray := normalizeImageLocalIllumination(targetImg, false)
	normPath, err := writeTempPNG(normGray, "sisaloja-norm-*.png")
	if err != nil {
		return "", err
	}
	defer os.Remove(normPath)

	// Pasada 1: Bloque uniforme (--psm 6) preservando columnas con espacios
	pass1, err1 := runTesseractArgs(bin, normPath, lang, "--oem", "1", "--psm", "6", "--dpi", "300", "-c", "preserve_interword_spaces=1")

	var parts []string
	if strings.TrimSpace(pass1) != "" {
		parts = append(parts, strings.TrimSpace(pass1))
	}

	// Pasada 2: Franja inferior adaptativa + 2x upscale dedicada para MRZ TD1 (IDBOL<<<)
	nb := normGray.Bounds()
	nw, nh := nb.Dx(), nb.Dy()
	mrzY0 := int(float64(nh) * 0.52)
	mrzY1 := nh
	if float64(nh) > float64(nw)*1.1 {
		// Foto vertical de celular: la tarjeta suele estar centrada entre 20% y 84%
		mrzY0 = int(float64(nh) * 0.42)
		mrzY1 = int(float64(nh) * 0.85)
	}
	if mrzY1-mrzY0 >= 40 && nw >= 100 {
		crop := cropImage(targetImg, 0, int(float64(targetImg.Bounds().Dy())*float64(mrzY0)/float64(nh)), targetImg.Bounds().Dx(), int(float64(targetImg.Bounds().Dy())*float64(mrzY1)/float64(nh)))
		cw, ch := crop.Bounds().Dx(), crop.Bounds().Dy()
		upW, upH := cw*2, ch*2
		if upW > 2200 {
			ratio := 2200.0 / float64(upW)
			upW = 2200
			upH = int(math.Round(float64(upH) * ratio))
		}
		upCrop := resizeBilinear(crop, upW, upH)
		mrzBin := normalizeImageLocalIllumination(upCrop, true)
		if mrzPath, errM := writeTempPNG(mrzBin, "sisaloja-mrz-*.png"); errM == nil {
			defer os.Remove(mrzPath)
			mrzOut, _ := runTesseractArgs(bin, mrzPath, "eng", "--oem", "1", "--psm", "6", "--dpi", "300", "-c", "tessedit_char_whitelist=ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789<")
			if strings.TrimSpace(mrzOut) == "" {
				mrzOut, _ = runTesseractArgs(bin, mrzPath, lang, "--oem", "1", "--psm", "6", "--dpi", "300", "-c", "tessedit_char_whitelist=ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789<")
			}
			if filteredMRZ := filterLikelyMRZLines(mrzOut); filteredMRZ != "" {
				parts = append(parts, filteredMRZ)
			}
		}
	}

	// Pasada 3 (rescate): Si Pasada 1 trajo muy poco texto o no detectó encabezados/MRZ, intentar --psm 11 (texto disperso)
	upperP1 := strings.ToUpper(pass1)
	hasKeyHints := strings.Contains(upperP1, "NOMBR") ||
		strings.Contains(upperP1, "APELLID") ||
		strings.Contains(upperP1, "IDBOL") ||
		strings.Contains(upperP1, "<<") ||
		strings.Contains(upperP1, "NACIDO")
	if !hasKeyHints || len(strings.TrimSpace(pass1)) < 45 {
		if pass3, err3 := runTesseractArgs(bin, normPath, lang, "--oem", "1", "--psm", "11", "--dpi", "300", "-c", "preserve_interword_spaces=1"); err3 == nil && strings.TrimSpace(pass3) != "" {
			parts = append(parts, strings.TrimSpace(pass3))
		}
	}

	combined := strings.TrimSpace(strings.Join(parts, "\n"))
	if combined == "" && err1 != nil {
		return "", err1
	}
	return combined, nil
}

func runTesseractArgs(bin, imgPath, lang string, extraArgs ...string) (string, error) {
	args := []string{imgPath, "stdout"}
	if lang != "" {
		args = append(args, "-l", lang)
	}
	args = append(args, extraArgs...)

	cmd := exec.Command(bin, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		if lang != "" {
			// Reintentar con idioma por defecto si el paquete de idioma especificado no está instalado
			fallbackArgs := append([]string{imgPath, "stdout"}, extraArgs...)
			cmdFallback := exec.Command(bin, fallbackArgs...)
			out.Reset()
			cmdFallback.Stdout = &out
			if err2 := cmdFallback.Run(); err2 == nil {
				return out.String(), nil
			}
		}
		return "", fmt.Errorf("error ejecutando tesseract: %w", err)
	}
	return out.String(), nil
}

func filterLikelyMRZLines(raw string) string {
	var out []string
	for _, line := range strings.Split(strings.ReplaceAll(raw, "\r", "\n"), "\n") {
		clean := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(line), " ", ""))
		if len(clean) < 16 {
			continue
		}
		if strings.Contains(clean, "<<") ||
			strings.Contains(clean, "IDBOL") ||
			strings.Contains(clean, "1DBOL") ||
			strings.Contains(clean, "IDB0L") ||
			strings.Contains(clean, "BOL<<") ||
			strings.HasPrefix(clean, "P<") ||
			strings.HasPrefix(clean, "I<") {
			out = append(out, clean)
		}
	}
	return strings.Join(out, "\n")
}

func writeTempPNG(img image.Image, pattern string) (string, error) {
	tmpFile, err := os.CreateTemp("", pattern)
	if err != nil {
		return "", err
	}
	path := tmpFile.Name()
	if err := png.Encode(tmpFile, img); err != nil {
		tmpFile.Close()
		os.Remove(path)
		return "", err
	}
	tmpFile.Close()
	return path, nil
}

func cropImage(src image.Image, x0, y0, x1, y1 int) image.Image {
	b := src.Bounds()
	if x0 < 0 {
		x0 = 0
	}
	if y0 < 0 {
		y0 = 0
	}
	if x1 > b.Dx() {
		x1 = b.Dx()
	}
	if y1 > b.Dy() {
		y1 = b.Dy()
	}
	w, h := x1-x0, y1-y0
	if w <= 0 || h <= 0 {
		return src
	}
	dst := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(x, y, src.At(b.Min.X+x0+x, b.Min.Y+y0+y))
		}
	}
	return dst
}

func resizeBilinear(src image.Image, dstW, dstH int) *image.Gray {
	if dstW < 1 {
		dstW = 1
	}
	if dstH < 1 {
		dstH = 1
	}
	b := src.Bounds()
	srcW, srcH := b.Dx(), b.Dy()
	luma := make([]float64, srcW*srcH)
	for y := 0; y < srcH; y++ {
		for x := 0; x < srcW; x++ {
			r, g, bl, _ := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			// RGBA devuelve [0..65535]; dividir entre 257 da [0..255]
			luma[y*srcW+x] = (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bl)) / 257.0
		}
	}

	dst := image.NewGray(image.Rect(0, 0, dstW, dstH))
	xRatio := float64(srcW) / float64(dstW)
	yRatio := float64(srcH) / float64(dstH)
	for y := 0; y < dstH; y++ {
		sy := (float64(y)+0.5)*yRatio - 0.5
		if sy < 0 {
			sy = 0
		}
		y0 := int(sy)
		y1 := y0 + 1
		if y1 >= srcH {
			y1 = srcH - 1
		}
		wy := sy - float64(y0)
		for x := 0; x < dstW; x++ {
			sx := (float64(x)+0.5)*xRatio - 0.5
			if sx < 0 {
				sx = 0
			}
			x0 := int(sx)
			x1 := x0 + 1
			if x1 >= srcW {
				x1 = srcW - 1
			}
			wx := sx - float64(x0)

			v00 := luma[y0*srcW+x0]
			v10 := luma[y0*srcW+x1]
			v01 := luma[y1*srcW+x0]
			v11 := luma[y1*srcW+x1]
			v := (v00*(1-wx)+v10*wx)*(1-wy) + (v01*(1-wx)+v11*wx)*wy
			if v < 0 {
				v = 0
			} else if v > 255 {
				v = 255
			}
			dst.SetGray(x, y, color.Gray{Y: uint8(math.Round(v))})
		}
	}
	return dst
}

// normalizeImageLocalIllumination aplana sombras, reflejos y fondos de color (bandera/wiphala/guilloché)
// usando imagen integral local sin destruir texto tenue ni binarizar bruscamente el fondo.
func normalizeImageLocalIllumination(src image.Image, adaptiveBinary bool) *image.Gray {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return image.NewGray(image.Rect(0, 0, 1, 1))
	}

	luma := make([]float64, w*h)
	sat := make([]float64, (w+1)*(h+1))
	for y := 0; y < h; y++ {
		rowSum := 0.0
		for x := 0; x < w; x++ {
			r, g, bl, _ := src.At(b.Min.X+x, b.Min.Y+y).RGBA()
			v := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bl)) / 257.0
			luma[y*w+x] = v
			rowSum += v
			sat[(y+1)*(w+1)+(x+1)] = sat[y*(w+1)+(x+1)] + rowSum
		}
	}

	minDim := w
	if h < minDim {
		minDim = h
	}
	radius := int(math.Round(float64(minDim) * 0.038))
	if radius < 14 {
		radius = 14
	}
	if radius > 60 {
		radius = 60
	}

	out := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		y0 := y - radius
		if y0 < 0 {
			y0 = 0
		}
		y1 := y + radius + 1
		if y1 > h {
			y1 = h
		}
		for x := 0; x < w; x++ {
			x0 := x - radius
			if x0 < 0 {
				x0 = 0
			}
			x1 := x + radius + 1
			if x1 > w {
				x1 = w
			}

			area := float64((x1 - x0) * (y1 - y0))
			sum := sat[y1*(w+1)+x1] - sat[y0*(w+1)+x1] - sat[y1*(w+1)+x0] + sat[y0*(w+1)+x0]
			localMean := sum / area
			val := luma[y*w+x]

			var px uint8
			if adaptiveBinary {
				if val < localMean*0.86 && (localMean-val) > 10 {
					px = 0
				} else {
					px = 255
				}
			} else {
				effMean := localMean
				if effMean < 45 {
					effMean = 45
				}
				flat := (val / effMean) * 210.0
				norm := ((flat - 48.0) / (196.0 - 48.0)) * 255.0
				if norm < 0 {
					norm = 0
				} else if norm > 255 {
					norm = 255
				}
				px = uint8(math.Round(norm))
			}
			out.SetGray(x, y, color.Gray{Y: px})
		}
	}
	return out
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
		"text": `Extrae los datos de este documento de identidad (Cédula de Bolivia, DNI o Pasaporte) y responde ÚNICAMENTE con un objeto JSON válido con las claves exactas: document_type ("CI"|"DNI"|"PASAPORTE"), document_number, full_name, first_name, last_name, nationality, gender ("M"|"F"), birth_date ("YYYY-MM-DD"), occupation, origin_city.`,
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

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash:generateContent?key=%s", strings.TrimSpace(apiKey))
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
