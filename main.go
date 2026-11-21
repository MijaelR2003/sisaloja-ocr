package main

import (
	"fmt"
	"log"
	"os"

	"github.com/MijaelR2003/sisaloja-ocr/internal/handler"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func NewApp() *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:   "SisAloja OCR Microservice (:8090)",
		BodyLimit: 15 * 1024 * 1024, // 15 MB para fotos de cámara móvil
	})

	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return true
		},
		AllowCredentials: true,
		AllowMethods:     "GET,POST,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization",
	}))

	h := handler.NewOCRHandler()

	app.Get("/health", h.Health)

	api := app.Group("/api/v1/ocr")
	api.Post("/scan", h.ScanDocument)
	api.Post("/parse-text", h.ParseText)

	return app
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8090"
	}

	app := NewApp()
	log.Printf("[sisaloja-ocr] Microservicio de OCR y lectura de carnets escuchando en http://localhost:%s/api/v1/ocr", port)
	if err := app.Listen(fmt.Sprintf(":%s", port)); err != nil {
		log.Fatalf("[sisaloja-ocr] Error al iniciar: %v", err)
	}
}
