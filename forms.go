package forms

import (
	"errors"
	"fmt"
	"html/template"
	"reflect"
	"strconv"
	"strings"

	"github.com/Nigel2392/forms/validators"
	"github.com/Nigel2392/router/v3/request"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

func NewValue(s string) *FormData {
	return &FormData{Val: []string{s}}
}

type Form struct {
	Fields      []FormElement
	Errors      FormErrors
	BeforeValid func(*request.Request, *Form) error
	AfterValid  func(*request.Request, *Form) error
}

func (f *Form) Validate() bool {
	var valid = true
	if f.Errors == nil {
		f.Errors = make(FormErrors, 0)
	}
	for _, field := range f.Fields {
		var err = field.Validate()
		if err != nil {
			valid = false
			f.Errors = append(f.Errors, FormError{
				Name:     field.GetName(),
				FieldErr: err,
			})
			field.AddError(err)
		}
	}
	return valid
}

func (f Form) AsP() template.HTML {
	var b strings.Builder
	for _, field := range f.Fields {
		if !field.HasLabel() {
			b.WriteString(`<p>`)
			b.WriteString(field.Label().String())
			b.WriteString("</p>")
		}
		b.WriteString(`<p>`)
		b.WriteString(field.Field().String())
		b.WriteString("</p>")
	}
	return template.HTML(b.String())
}

func (f *Form) Fill(r *request.Request) bool {
	var err error
	r.Request.ParseForm()

	switch r.Method() {
	case "GET", "HEAD", "DELETE":
		f.fillQueries(r)
	case "POST", "PUT", "PATCH":
		f.fillForm(r)
	}

	if f.BeforeValid != nil {
		err = f.BeforeValid(r, f)
		if err != nil {
			f.AddError("Validation", err)
			return false
		}
	}

	valid := f.Validate()

	if f.AfterValid != nil && valid {
		err = f.AfterValid(r, f)
		if err != nil {
			f.AddError("Validation", err)
			return false
		}
	}

	return valid
}

func (f *Form) fillQueries(r *request.Request) {
	for _, field := range f.Fields {
		field.SetValue(r.Request.Form[field.GetName()])
	}
}

func (f *Form) fillForm(r *request.Request) {
	for _, field := range f.Fields {
		if field.IsFile() {
			var mForm = r.Request.MultipartForm
			if mForm == nil {
				continue
			}
			if mForm.File == nil {
				continue
			}
			var readerClosers = mForm.File[field.GetName()]
			if len(readerClosers) == 0 {
				continue
			}
			var readerCloser = readerClosers[0]
			var file, err = readerCloser.Open()
			if err != nil {
				f.AddError(field.GetName(), err)
			}
			field.SetFile(readerCloser.Filename, file)
			continue
		}
		field.SetValue(r.Request.PostForm[field.GetName()])
	}
}

func (f *Form) Clear() {
	for _, field := range f.Fields {
		field.Clear()
	}
}

func (f *Form) Field(name string) FormElement {
	for _, field := range f.Fields {
		if field.GetName() == name {
			return field
		}
	}
	return nil
}

// AddField adds a field to the form
func (f *Form) AddFields(field ...FormElement) {
	if f.Fields == nil {
		f.Fields = make([]FormElement, 0)
	}
	f.Fields = append(f.Fields, field...)
}

// AddError adds an error to the form
func (f *Form) AddError(name string, err error) {
	if f.Errors == nil {
		f.Errors = make(FormErrors, 0)
	}
	f.Errors = append(f.Errors, FormError{
		Name:     name,
		FieldErr: err,
	})
}

func (f *Form) Without(names ...string) {
	var fields = make([]FormElement, 0)
	for _, field := range f.Fields {
		var found = false
		for _, name := range names {
			if strings.EqualFold(field.GetName(), name) {
				found = true
				break
			}
		}
		if !found {
			fields = append(fields, field)
		}
	}
	f.Fields = fields
}

func (f *Form) Disabled(names ...string) Form {
	if len(names) == 0 {
		for _, field := range f.Fields {
			field.SetDisabled(true)
		}
		return *f
	}
	for _, field := range f.Fields {
		for _, name := range names {
			if strings.EqualFold(field.GetName(), name) {
				field.SetDisabled(true)
				break
			}
		}
	}
	return *f
}

func (f *Form) Get(name string) *FormData {
	for _, field := range f.Fields {
		if field.GetName() == name {
			return field.Value()
		}
	}
	return nil
}

var DefaultTitleCaser = cases.Title(language.English).String

func (f *Form) CSRFToken(csrf_token string) *Form {
	var field = newField(TypeHidden, "csrf_token", "csrf_token", "", "", csrf_token)
	field.LabelText = ""
	f.AddFields(field)
	return f
}

func (f *Form) TextField(name string, id string, classes string, placeholder string, value string) *Field {
	var field = newField(TypeText, name, id, classes, placeholder, value)
	f.AddFields(field)
	return field
}

func (f *Form) PasswordField(name string, id string, classes string, placeholder string, value string) *Field {
	var field = newField(TypePassword, name, id, classes, placeholder, value)
	f.AddFields(field)
	return field
}

func (f *Form) EmailField(name string, id string, classes string, placeholder string, value string) *Field {
	var field = newField(TypeEmail, name, id, classes, placeholder, value)
	field.Validators = validators.New(
		validators.Email,
	)
	f.AddFields(field)
	return field
}

func (f *Form) NumberField(name string, id string, classes string, placeholder string, value int) *Field {
	var v = strconv.Itoa(value)
	var field = newField(TypeNumber, name, id, classes, placeholder, v)
	f.AddFields(field)
	return field
}

func (f *Form) FileField(name string, id string, classes string, placeholder string, path string) *Field {
	var field = newField(TypeFile, name, id, classes, placeholder, "")
	field.LabelText = path
	f.AddFields(field)
	return field
}

func (f *Form) HiddenField(name string, id string, classes string, placeholder string, value string) *Field {
	var field = newField(TypeHidden, name, id, classes, placeholder, value)
	f.AddFields(field)
	return field
}

func (f *Form) TextAreaField(name string, id string, classes string, placeholder string, value string) *Field {
	var field = newField(TypeTextArea, name, id, classes, placeholder, value)
	f.AddFields(field)
	return field
}

func (f *Form) SelectField(name string, id string, classes string, options []Option) *Field {
	var field = newField(TypeSelect, name, id, classes, "", "")
	field.Options = options
	f.AddFields(field)
	return field
}

func (f *Form) CheckboxField(name string, id string, classes string, placeholder string, value bool) *Field {
	var field = newField(TypeCheck, name, id, classes, placeholder, "")
	field.SetChecked(value)
	f.AddFields(field)
	return field
}

func (f *Form) RadioField(name string, id string, classes string, placeholder string, value bool) *Field {
	var field = newField(TypeRadio, name, id, classes, placeholder, "")
	field.Checked = value
	f.AddFields(field)
	return field
}

func (f *Form) SubmitButton(name string, id string, classes string, value string) *Field {
	var field = newField(TypeSubmit, name, id, classes, "", value)
	f.AddFields(field)
	return field
}

func (f *Form) ResetButton(name string, id string, classes string, value string) *Field {
	var field = newField(TypeReset, name, id, classes, "", value)
	f.AddFields(field)
	return field
}

func (f *Form) Button(name string, id string, classes string, value string) *Field {
	var field = newField(TypeButton, name, id, classes, "", value)
	f.AddFields(field)
	return field
}

// Any field which is not a primitive type or a slice of a primitive type must implement this interface to be scanned
//
// The field must be able to scan a string into itself
type Scanner interface {
	ScanStr(string) error
}

// Valuer returns the underlying value represented as a string.
type Valuer interface {
	StringValue() string
}

// Scan scans the form data into the form fields
//
// Otherwise, the fields are scanned in the order they are provided.
//
// # The fields are matched by it's GetName() method, case insensitive
//
// If fields is ["*"] or len(fields) == 0, all fields are scanned
func (f *Form) Scan(fields []string, data ...any) error {
	var isAllFields = false
	if len(fields) != len(data) {
		if len(fields) >= 1 && fields[0] == "*" {
			isAllFields = true
		} else if len(fields) == 0 {
			isAllFields = true
		} else {
			return fmt.Errorf("fields and data must be of same length, otherwise fields must be '*' or empty")
		}
	}
	var fieldsInOrder []FormElement
	if isAllFields {
		fieldsInOrder = f.Fields
	} else {
		fieldsInOrder = make([]FormElement, 0, len(fields))
		for _, field := range fields {
		inner:
			for _, f := range f.Fields {
				if strings.EqualFold(f.GetName(), field) {
					fieldsInOrder = append(fieldsInOrder, f)
					break inner
				}
			}
		}
	}

	// Verify that the data and fields lengths are the same again.
	if len(fieldsInOrder) != len(data) {
		return fmt.Errorf("Length mismatch between fields and data")
	}

	for i, field := range fieldsInOrder {
		var v = field.Value()
		if v == nil {
			continue
		}
		var scanInto = data[i]
		var reflectOf = reflect.ValueOf(scanInto)
		if reflectOf.Kind() != reflect.Ptr {
			return fmt.Errorf("data must be a pointer")
		}
		var fieldVal = field.Value().Value()
		var fieldValStr string
		if len(fieldVal) == 0 {
			continue
		}
		fieldValStr = fieldVal[0]
		var reflectElem = reflectOf.Elem()
		switch reflectElem.Kind() {
		case reflect.String:
			reflectElem.SetString(fieldValStr)
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			var val, err = strconv.ParseInt(fieldValStr, 10, 64)
			if err != nil {
				return errors.New("invalid integer")
			}
			reflectElem.SetInt(val)
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
			var val, err = strconv.ParseUint(fieldValStr, 10, 64)
			if err != nil {
				return errors.New("invalid unsigned integer")
			}
			reflectElem.SetUint(val)
		case reflect.Float32, reflect.Float64:
			var val, err = strconv.ParseFloat(fieldValStr, 64)
			if err != nil {
				return errors.New("invalid float")
			}
			reflectElem.SetFloat(val)
		case reflect.Bool:
			var val, err = parseBool(fieldValStr)
			if err != nil {
				return errors.New("invalid boolean")
			}
			reflectElem.SetBool(val)
		case reflect.Slice:
			var elemTyp = reflectElem.Type().Elem()
			switch elemTyp.Kind() {
			case reflect.String:
				reflectElem.Set(reflect.ValueOf(fieldVal))
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				var val = make([]int64, 0, len(fieldVal))
				for _, v := range fieldVal {
					var i, err = strconv.ParseInt(v, 10, 64)
					if err != nil {
						return errors.New("invalid integer")
					}
					val = append(val, i)
				}
				reflectElem.Set(reflect.ValueOf(val))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				var val = make([]uint64, 0, len(fieldVal))
				for _, v := range fieldVal {
					var i, err = strconv.ParseUint(v, 10, 64)
					if err != nil {
						return errors.New("invalid unsigned integer")
					}
					val = append(val, i)
				}
				reflectElem.Set(reflect.ValueOf(val))
			case reflect.Float32, reflect.Float64:
				var val = make([]float64, 0, len(fieldVal))
				for _, v := range fieldVal {
					var i, err = strconv.ParseFloat(v, 64)
					if err != nil {
						return errors.New("invalid float")
					}
					val = append(val, i)
				}
				reflectElem.Set(reflect.ValueOf(val))
			case reflect.Bool:
				var val = make([]bool, 0, len(fieldVal))
				for _, v := range fieldVal {
					var i, err = parseBool(v)
					if err != nil {
						return errors.New("invalid boolean")
					}
					val = append(val, i)
				}
				reflectElem.Set(reflect.ValueOf(val))
			default:
				return fmt.Errorf("invalid slice type type, %s", reflectElem.Kind().String())
			}
		default:
			var vInterface = reflectOf.Interface()
			var converter, ok = vInterface.(Scanner)
			if !ok {
				return fmt.Errorf("invalid field type, %s", reflectElem.Kind().String())
			}
			var err = converter.ScanStr(fieldValStr)
			if err != nil {
				return fmt.Errorf("invalid value, %s", err.Error())
			}
		}
	}
	return nil
}

func newField(typ string, name string, id string, classes string, placeholder string, value string) *Field {
	var field = &Field{
		Type:        typ,
		LabelText:   DefaultTitleCaser(name),
		Name:        name,
		ID:          id,
		Class:       classes,
		Placeholder: placeholder,
		FormValue:   NewValue(value),
	}
	return field
}

func parseBool(s string) (bool, error) {
	switch strings.ToLower(s) {
	case "true", "yes", "1", "on", "checked", "selected":
		return true, nil
	case "false", "no", "0":
		return false, nil
	}
	return false, fmt.Errorf("could not parse bool")
}
