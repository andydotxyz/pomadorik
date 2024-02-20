//go:generate fyne bundle -package icon -o icondata.go .

package icon

var (
	Data     = resourceAppIconPng
	Disabled = resourceDisabledPng
)
