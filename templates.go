package ralph

import "embed"

// TemplatesFS contains all embedded template files.
//
//go:embed all:templates
var TemplatesFS embed.FS
