package tui

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/BetaLixT/podsync/internal/config"
	"github.com/charmbracelet/bubbles/textinput"
)

type optionsCtrl struct {
	config      *config.Config
	configType  reflect.Type
	configValue reflect.Value
	cursor      int
	height      int
	offset      int
	textInput   textinput.Model
	insertMode  bool
}

func newOptionsCtrl(cfg config.Config) *optionsCtrl {
	return &optionsCtrl{
		config:      &cfg,
		configType:  reflect.TypeOf(config.Config{}),
		configValue: reflect.ValueOf(&cfg).Elem(),
		cursor:      0,
		height:      10,
		offset:      0,
	}
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

func (o *optionsCtrl) InsertMode(ti textinput.Model) textinput.Model {
	o.insertMode = true
	o.textInput = ti
	o.textInput.Placeholder = getOptionValue(o.configValue.Field(o.cursor))
	o.textInput.Focus()
	o.textInput.CharLimit = 156
	o.textInput.Width = 20
	return o.textInput
}

func (o *optionsCtrl) InsertModeUpdate(ti textinput.Model) textinput.Model {
	o.textInput = ti
	return o.textInput
}

func (o *optionsCtrl) InsertModeSave() {
	o.insertMode = false
}

func (o *optionsCtrl) InsertModeCancel() {
	o.insertMode = false
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
		line := o.formatOptionLine(name, value, width, i == o.cursor, o.insertMode)
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
		value = o.textInput.View()
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
