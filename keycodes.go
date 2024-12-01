package term

// Taken from tcell
const (
	CtrlSpace = iota
	CtrlA
	CtrlB
	CtrlC
	CtrlD
	CtrlE
	CtrlF
	CtrlG
	CtrlH // Backspace
	CtrlI // Tab
	CtrlJ
	CtrlK
	CtrlL
	CtrlM // Return
	CtrlN
	CtrlO
	CtrlP
	CtrlQ
	CtrlR
	CtrlS
	CtrlT
	CtrlU
	CtrlV
	CtrlW
	CtrlX
	CtrlY
	CtrlZ
	CtrlLeftSq // Escape
	CtrlBackslash
	CtrlRightSq
	CtrlCarat
	CtrlUnderscore
)

const (
	Backspace = 8
	Tab       = 9
	Return    = 13
	Esc       = 27
)

// also from tcell
var KeyNames = map[byte]string{
	Return:         "Return",
	Backspace:      "Backspace",
	Tab:            "Tab",
	Esc:            "Esc",
	CtrlA:          "Ctrl-A",
	CtrlB:          "Ctrl-B",
	CtrlC:          "Ctrl-C",
	CtrlD:          "Ctrl-D",
	CtrlE:          "Ctrl-E",
	CtrlF:          "Ctrl-F",
	CtrlG:          "Ctrl-G",
	CtrlJ:          "Ctrl-J",
	CtrlK:          "Ctrl-K",
	CtrlL:          "Ctrl-L",
	CtrlN:          "Ctrl-N",
	CtrlO:          "Ctrl-O",
	CtrlP:          "Ctrl-P",
	CtrlQ:          "Ctrl-Q",
	CtrlR:          "Ctrl-R",
	CtrlS:          "Ctrl-S",
	CtrlT:          "Ctrl-T",
	CtrlU:          "Ctrl-U",
	CtrlV:          "Ctrl-V",
	CtrlW:          "Ctrl-W",
	CtrlX:          "Ctrl-X",
	CtrlY:          "Ctrl-Y",
	CtrlZ:          "Ctrl-Z",
	CtrlSpace:      "Ctrl-Space",
	CtrlUnderscore: "Ctrl-_",
	CtrlRightSq:    "Ctrl-]",
	CtrlBackslash:  "Ctrl-\\",
	CtrlCarat:      "Ctrl-^",
}
