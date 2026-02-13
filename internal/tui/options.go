package tui

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/BetaLixT/podsync/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type optionsCtrl struct {
	config      interface{}
	configType  reflect.Type
	configValue reflect.Value
	cursor      int
	height      int
	offset      int
	input       *optionInput
}

type optionInput struct {
	textInput textinput.Model
	fieldKind reflect.Kind
}

func newOptionInput(
	textInput textinput.Model,
	fieldKind reflect.Kind,
) *optionInput {
	return &optionInput{
		textInput,
		fieldKind,
	}
}

func newOptionsCtrl[T any](cfg T) *optionsCtrl {
	return &optionsCtrl{
		config:      &cfg,
		configType:  reflect.TypeOf(config.Config{}),
		configValue: reflect.ValueOf(&cfg).Elem(),
		cursor:      0,
		height:      10,
		offset:      0,
	}
}

func getConfig[T any](optCtrl *optionsCtrl) (t T, err error) {
	x, ok := optCtrl.config.(T)
	if !ok {
		return t, fmt.Errorf("failed to cast")
	}
	return x, nil
}

// func (o *optionsView) SetConfig(cfg *config.Config) {
// 	o.config = cfg
// 	o.configValue = reflect.ValueOf(cfg).Elem()
// 	o.updateOffset()
// }

func (o *optionsCtrl) SetHeight(h int) {
	o.height = h
	o.updateOffset()
}

func (o *optionsCtrl) MoveUp() {
	if o.cursor > 0 {
		o.cursor--
		o.updateOffset()
	}
}

func (o *optionsCtrl) MoveDown() {
	if o.cursor < o.configType.NumField()-1 {
		o.cursor++
		o.updateOffset()
	}
}

func (o *optionsCtrl) InsertMode() textinput.Model {
	field := o.configValue.Field(o.cursor)
	fieldKind := field.Kind()

	ti := textinput.New()
	ti.Placeholder = getOptionValue(field)
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20

	switch fieldKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		ti.Validate = func(s string) error {
			if s == "" {
				return nil
			}
			_, err := strconv.ParseInt(s, 10, 64)
			return err
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		ti.Validate = func(s string) error {
			if s == "" {
				return nil
			}
			_, err := strconv.ParseUint(s, 10, 64)
			return err
		}
	case reflect.Float32, reflect.Float64:
		ti.Validate = func(s string) error {
			if s == "" {
				return nil
			}
			_, err := strconv.ParseFloat(s, 64)
			return err
		}
	case reflect.Bool:
		ti.Validate = func(s string) error {
			if strings.HasPrefix("true", s) || strings.HasPrefix("false", s) {
				return nil
			}
			return fmt.Errorf("not true/false ")
		}
	default:
		ti.Validate = func(s string) error {
			return nil
		}
	}

	o.input = newOptionInput(ti, fieldKind)
	return o.input.textInput
}

func (o *optionsCtrl) InsertModeUpdate(
	msg tea.Msg,
) (textinput.Model, tea.Cmd) {
	// var cmd tea.Cmd
	ti, cmd := o.input.textInput.Update(msg)

	if ti.Validate(ti.Value()) != nil {
		// TODO: alert the ommission
		// println("noval")
		return o.input.textInput, nil
	}

	o.input.textInput = ti
	return o.input.textInput, cmd
}

func (o *optionsCtrl) InsertModeSave() error {

	valStr := o.input.textInput.Value()
	defer func() {
		o.input = nil
	}()

	field := o.configValue.Field(o.cursor)

	switch o.input.fieldKind {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		val, err := strconv.ParseInt(valStr, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(val)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		val, err := strconv.ParseUint(valStr, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(val)
		return nil
	case reflect.Float32, reflect.Float64:
		val, err := strconv.ParseFloat(valStr, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(val)
		return nil
	case reflect.Bool:
		val, err := strconv.ParseBool(valStr)
		if err != nil {
			return err
		}
		field.SetBool(val)
		return nil
	case reflect.String:
		field.SetString(valStr)
		return nil
	default:
		return fmt.Errorf("unhandled type")
	}
}

func (o *optionsCtrl) InsertModeCancel() {
	o.input = nil
}

func (o *optionsCtrl) updateOffset() {
	if o.cursor < o.offset {
		o.offset = o.cursor
	} else if o.cursor >= o.offset+o.height {
		o.offset = o.cursor - o.height + 1
	}
}

func (o *optionsCtrl) Count() int {
	return o.configType.NumField()
}

func (o *optionsCtrl) View(width int) string {

	if o.configType.NumField() == 0 {
		return subtitleStyle.Render("  No options available.")
	}

	visibleEnd := min(o.offset+o.height, o.configType.NumField())

	var b strings.Builder
	b.WriteString(subtitleStyle.Render("Options"))
	b.WriteString("\n")
	for i := o.offset; i < visibleEnd; i++ {
		name := getOptionName(o.configType.Field(i))
		value := getOptionValue(o.configValue.Field(i))
		line := o.formatOptionLine(name, value, width, i == o.cursor, o.input != nil)
		b.WriteString(line)
		if i < visibleEnd-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (o *optionsCtrl) formatOptionLine(name, value string, width int, selected, insertMode bool) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}

	availableWidth := width - len(prefix) - 3
	if availableWidth < 20 {
		availableWidth = 20
	}

	nameWidth := min(25, availableWidth/3)
	valueWidth := availableWidth - nameWidth - 3

	name = truncate(name, nameWidth)

	if selected && insertMode {
		value = o.input.textInput.View()
	} else {
		value = truncate(value, valueWidth)
	}

	name = padRight(name, nameWidth)

	line := prefix + name + " = " + value

	if selected {
		return selectedStyle.Render(line)
	}
	return normalStyle.Render(line)
}

func getOptionName(field reflect.StructField) string {
	val, ok := field.Tag.Lookup("yaml")
	if ok {
		return val
	}

	val, ok = field.Tag.Lookup("json")
	if ok {
		return val
	}

	return field.Name
}

func getOptionValue(field reflect.Value) string {
	switch field.Kind() {
	case reflect.String:
		return field.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", field.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", field.Uint())
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%g", field.Float())
	case reflect.Bool:
		if field.Bool() {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprintf("%v", field.Interface())
	}
}
