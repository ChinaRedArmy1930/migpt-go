package doc

import _ "embed"

var (
	//go:embed template/default_system_template.tmpl
	DefaultSystemTemplate string

	//go:embed template/tool_gen.tmpl
	ToolGenTemplate string
)
