package tools

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// CalculatorTool provides basic arithmetic operations
type CalculatorTool struct{}

func (c *CalculatorTool) Name() string {
	return "calculator"
}

func (c *CalculatorTool) Description() string {
	return "Perform basic arithmetic operations (add, subtract, multiply, divide)"
}

func (c *CalculatorTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"operation": map[string]interface{}{
				"type":        "string",
				"description": "The operation to perform",
				"enum":        []string{"add", "subtract", "multiply", "divide"},
			},
			"a": map[string]interface{}{
				"type":        "number",
				"description": "First number",
			},
			"b": map[string]interface{}{
				"type":        "number",
				"description": "Second number",
			},
		},
		"required": []string{"operation", "a", "b"},
	}
}

func (c *CalculatorTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	operation, ok := args["operation"].(string)
	if !ok {
		return nil, fmt.Errorf("operation must be a string")
	}

	a, err := c.parseNumber(args["a"])
	if err != nil {
		return nil, fmt.Errorf("invalid number a: %v", err)
	}

	b, err := c.parseNumber(args["b"])
	if err != nil {
		return nil, fmt.Errorf("invalid number b: %v", err)
	}

	var result float64
	switch operation {
	case "add":
		result = a + b
	case "subtract":
		result = a - b
	case "multiply":
		result = a * b
	case "divide":
		if b == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		result = a / b
	default:
		return nil, fmt.Errorf("unsupported operation: %s", operation)
	}

	return map[string]interface{}{
		"result":    result,
		"operation": operation,
		"operands":  []float64{a, b},
	}, nil
}

func (c *CalculatorTool) parseNumber(v interface{}) (float64, error) {
	switch num := v.(type) {
	case float64:
		return num, nil
	case float32:
		return float64(num), nil
	case int:
		return float64(num), nil
	case int32:
		return float64(num), nil
	case int64:
		return float64(num), nil
	case string:
		return strconv.ParseFloat(num, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to number", v)
	}
}

// TimeTool provides current time information
type TimeTool struct{}

func (t *TimeTool) Name() string {
	return "current_time"
}

func (t *TimeTool) Description() string {
	return "Get current time in various formats"
}

func (t *TimeTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"format": map[string]interface{}{
				"type":        "string",
				"description": "Time format (iso, unix, rfc3339, or custom layout)",
				"default":     "iso",
			},
			"timezone": map[string]interface{}{
				"type":        "string",
				"description": "Timezone (e.g., UTC, America/New_York)",
				"default":     "UTC",
			},
		},
	}
}

func (t *TimeTool) Execute(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	format := "iso"
	if f, ok := args["format"].(string); ok && f != "" {
		format = f
	}

	timezone := "UTC"
	if tz, ok := args["timezone"].(string); ok && tz != "" {
		timezone = tz
	}

	// Load timezone
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("invalid timezone: %v", err)
	}

	now := time.Now().In(loc)

	var formatted string
	switch format {
	case "iso":
		formatted = now.Format("2006-01-02T15:04:05Z07:00")
	case "unix":
		formatted = strconv.FormatInt(now.Unix(), 10)
	case "rfc3339":
		formatted = now.Format(time.RFC3339)
	default:
		// Custom format
		formatted = now.Format(format)
	}

	return map[string]interface{}{
		"timestamp":  formatted,
		"timezone":   timezone,
		"unix":       now.Unix(),
		"year":       now.Year(),
		"month":      int(now.Month()),
		"day":        now.Day(),
		"hour":       now.Hour(),
		"minute":     now.Minute(),
		"second":     now.Second(),
		"weekday":    now.Weekday().String(),
	}, nil
}