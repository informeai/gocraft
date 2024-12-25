package shader

import _ "embed"

var (
	//go:embed block.vert
	BlockVertexSource string

	//go:embed block.frag
	BlockFragmentSource string

	//go:embed line.vert
	LineVertexSource string

	//go:embed line.frag
	LineFragmentSource string

	//go:embed player.vert
	PlayerVertexSource string

	//go:embed player.frag
	PlayerFragmentSource string
)
