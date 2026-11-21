# 🪪 sisaloja-ocr

Microservicio HTTP independiente de **reconocimiento y estructuración de documentos de identidad** (Cédula de Identidad de Bolivia **SEGIP** nueva y clásica, **DNI** de Perú/Argentina/Chile, **Pasaportes** y lectura matemática de zonas **MRZ ICAO 9303** `TD1`/`TD3`).

> [!IMPORTANT]
> Este microservicio es **100% autónomo y sin base de datos** (*stateless*). Expone una API REST en el puerto **`:8090`** (`/api/v1/ocr/*`) y también es enrutado por el servidor principal `sisaloja-backend` (`:8000/api/v1/ocr/scan`). La especificación completa está en el repositorio central **[sisaloja](https://github.com/MijaelR2003/sisaloja)** (`docs/01`, `docs/03` y `docs/06`).

---

## 🚀 Endpoints Expuestos (`:8090`)

| Método y Ruta | Descripción |
| :--- | :--- |
| `GET /health` | Estado del microservicio, disponibilidad de motores (`mrz_icao9303`, `segip_bolivia`, `tesseract_cli`, `gemini_vision`). |
| `POST /api/v1/ocr/scan` | Recibe imágenes (`front_image`, `back_image` en `multipart/form-data` o Base64) y/o texto OCR pre-extraído (`front_text`, `back_text`) y devuelve los campos estructurados del huésped listos para el formulario de Check-in o Reserva. |
| `POST /api/v1/ocr/parse-text` | Decodifica y fusiona bloques de texto crudo (frente + reverso + líneas MRZ `<<<`) aplicando corrección de errores ópticos y reglas SEGIP/ICAO 9303. |

---

## 📦 Ejemplo de Petición y Respuesta (`POST /api/v1/ocr/scan`)

### Entrada (JSON o `multipart/form-data`):
```json
{
  "front_text": "ESTADO PLURINACIONAL DE BOLIVIA SEGIP CEDULA DE IDENTIDAD N° 8392011 LP NOMBRES: ALEJANDRO APELLIDOS: TORREZ SILVA FECHA DE NACIMIENTO: 14 DE MAYO DE 1992",
  "back_text": "LUGAR DE NACIMIENTO: MURILLO - LA PAZ OCUPACION / PROFESION: INGENIERO DE SISTEMAS IDBOL8392011<<4<<<<<<<<<<<<<<< 9205148M3011205BOL<<<<<<<<<<<6 TORREZ<SILVA<<ALEJANDRO<<<<<<<"
}
```
*(También acepta `front_image` y `back_image` como archivos `multipart/form-data` o DataURL Base64).*

### Respuesta (`200 OK`):
```json
{
  "service": "sisaloja-ocr",
  "engine": "hybrid-mrz-segip",
  "confidence": "ALTA (MRZ Verificado)",
  "detected_mrz": true,
  "fields": {
    "document_type": "CI",
    "document_number": "8392011 LP",
    "full_name": "Alejandro Torrez Silva",
    "nationality": "Boliviana",
    "gender": "M",
    "birth_date": "1992-05-14",
    "occupation": "Ingeniero De Sistemas",
    "origin_city": "La Paz"
  }
}
```

---

## 🛠️ Ejecución en Local y Pruebas

```bash
cp .env.example .env

# Levantar el microservicio en http://localhost:8090
go run .

# Ejecutar pruebas automatizadas (Frente, Reverso, MRZ TD1/TD3 y Endpoints HTTP)
go test -v ./...
```
