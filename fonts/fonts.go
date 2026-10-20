// Package fonts embeds the typeface used for text overlays, so the binary
// does not depend on where it is run from.
package fonts

import _ "embed"

// NotoSansRegular is Noto Sans Regular (SIL Open Font License) as TrueType.
//
//go:embed NotoSans-Regular.ttf
var NotoSansRegular []byte
