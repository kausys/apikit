package codegen

import (
	"strings"
	"testing"

	"github.com/kausys/apikit/parser"
)

func TestGenerate_FiberFramework(t *testing.T) {
	gen, err := NewWithFramework("fiber")
	if err != nil {
		t.Fatalf("NewWithFramework failed: %v", err)
	}

	result := &parser.ParseResult{
		Source: parser.Source{Package: "test"},
		Handlers: []parser.Handler{{
			Name:       "GetUser",
			ParamType:  "GetUserRequest",
			ReturnType: "GetUserResponse",
			Struct:     &parser.Struct{},
		}},
	}

	code, err := gen.Generate(result)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)
	t.Log("Generated code:\n", codeStr)

	// Check for Fiber imports
	if !strings.Contains(codeStr, `"github.com/gofiber/fiber/v2"`) {
		t.Error("Expected fiber import")
	}
	if !strings.Contains(codeStr, `"github.com/gofiber/fiber/v2/middleware/adaptor"`) {
		t.Error("Expected adaptor import")
	}

	// Check for fiber.Handler return type
	if !strings.Contains(codeStr, "fiber.Handler") {
		t.Error("Expected fiber.Handler return type")
	}

	// Check for adaptor wrapper
	if !strings.Contains(codeStr, "adaptor.HTTPHandlerFunc") {
		t.Error("Expected adaptor.HTTPHandlerFunc wrapper")
	}
}

func TestGenerate_GinFramework(t *testing.T) {
	gen, err := NewWithFramework("gin")
	if err != nil {
		t.Fatalf("NewWithFramework failed: %v", err)
	}

	result := &parser.ParseResult{
		Source: parser.Source{Package: "test"},
		Handlers: []parser.Handler{{
			Name:       "GetUser",
			ParamType:  "GetUserRequest",
			ReturnType: "GetUserResponse",
			Struct:     &parser.Struct{},
		}},
	}

	code, err := gen.Generate(result)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)
	t.Log("Generated code:\n", codeStr)

	// Check for Gin imports
	if !strings.Contains(codeStr, `"github.com/gin-gonic/gin"`) {
		t.Error("Expected gin import")
	}

	// Check for gin.HandlerFunc return type
	if !strings.Contains(codeStr, "gin.HandlerFunc") {
		t.Error("Expected gin.HandlerFunc return type")
	}

	// Check for gin.WrapF wrapper (for http.HandlerFunc)
	if !strings.Contains(codeStr, "gin.WrapF") {
		t.Error("Expected gin.WrapF wrapper")
	}
}

func TestGenerate_EchoFramework(t *testing.T) {
	gen, err := NewWithFramework("echo")
	if err != nil {
		t.Fatalf("NewWithFramework failed: %v", err)
	}

	result := &parser.ParseResult{
		Source: parser.Source{Package: "test"},
		Handlers: []parser.Handler{{
			Name:       "GetUser",
			ParamType:  "GetUserRequest",
			ReturnType: "GetUserResponse",
			Struct:     &parser.Struct{},
		}},
	}

	code, err := gen.Generate(result)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	codeStr := string(code)
	t.Log("Generated code:\n", codeStr)

	// Check for Echo imports
	if !strings.Contains(codeStr, `"github.com/labstack/echo/v4"`) {
		t.Error("Expected echo import")
	}

	// Check for echo.HandlerFunc return type
	if !strings.Contains(codeStr, "echo.HandlerFunc") {
		t.Error("Expected echo.HandlerFunc return type")
	}

	// Check for echo.WrapHandler wrapper
	if !strings.Contains(codeStr, "echo.WrapHandler") {
		t.Error("Expected echo.WrapHandler wrapper")
	}
}
