package prompt

var keyMap = map[string]rune{
	// Key values from the MDN's UI Events API[1], but only those having X11 symbols and codes[2],
	// because that's what VNC (RFB) protocol uses
	//
	// [1]: https://developer.mozilla.org/en-US/docs/Web/API/UI_Events/Keyboard_event_key_values
	// [2]: https://gitlab.freedesktop.org/xorg/proto/xorgproto/-/blob/master/include/X11/keysymdef.h

	// Modifier keys
	"alt":        0xffe9, // XK_Alt_L
	"altgraph":   0xff7e, // XK_Mode_switch
	"capslock":   0xffe5, // XK_Caps_Lock
	"control":    0xffe3, // XK_Control_L
	"hyper":      0xffed, // XK_Hyper_L
	"meta":       0xffe7, // XK_Meta_L
	"numlock":    0xff7f, // XK_Num_Lock
	"scrolllock": 0xff14, // XK_Scroll_Lock
	"shift":      0xffe1, // XK_Shift_L
	"super":      0xffeb, // XK_Super_L

	// Whitespace keys
	"enter": 0xff0d, // XK_Return
	"tab":   0xff09, // XK_Tab

	// Navigation keys
	"arrowdown":  0xff54, // XK_Down
	"arrowleft":  0xff51, // XK_Left
	"arrowright": 0xff53, // XK_Right
	"arrowup":    0xff52, // XK_Up
	"end":        0xff57, // XK_End
	"home":       0xff50, // XK_Home
	"pagedown":   0xff56, // XK_Page_Down
	"pageup":     0xff55, // XK_Page_Up

	// Editing keys
	"backspace": 0xff08, // XK_BackSpace
	"clear":     0xff0b, // XK_Clear
	"crsel":     0xfd1c, // XK_3270_CursorSelect
	"delete":    0xffff, // XK_Delete
	"eraseeof":  0xfd1b, // XK_3270_ExSelect
	"exsel":     0xfd1b, // XK_3270_ExSelect
	"insert":    0xff63, // XK_Insert
	"redo":      0xff66, // XK_Redo
	"undo":      0xff65, // XK_Undo

	// UI keys
	"again":       0xff66, // XK_Redo
	"attn":        0xfd0e, // XK_3270_Attn
	"cancel":      0xff69, // XK_Cancel
	"contextmenu": 0xff67, // XK_Menu
	"escape":      0xff1b, // XK_Escape
	"find":        0xff68, // XK_Find
	"help":        0xff6a, // XK_Help
	"pause":       0xff13, // XK_Pause
	"play":        0xfd16, // XK_3270_Play
	"print":       0xff61, // XK_Print
	"printscreen": 0xfd1d, // XK_3270_PrintScreen
	"select":      0xff60, // XK_Select

	// Function keys
	"f1":  0xffbe, // XK_F1
	"f2":  0xffbf, // XK_F2
	"f3":  0xffc0, // XK_F3
	"f4":  0xffc1, // XK_F4
	"f5":  0xffc2, // XK_F5
	"f6":  0xffc3, // XK_F6
	"f7":  0xffc4, // XK_F7
	"f8":  0xffc5, // XK_F8
	"f9":  0xffc6, // XK_F9
	"f10": 0xffc7, // XK_F10
	"f11": 0xffc8, // XK_F11
	"f12": 0xffc9, // XK_F12
	"f13": 0xffca, // XK_F13
	"f14": 0xffcb, // XK_F14
	"f15": 0xffcc, // XK_F15
	"f16": 0xffcd, // XK_F16
	"f17": 0xffce, // XK_F17
	"f18": 0xffcf, // XK_F18
	"f19": 0xffd0, // XK_F19
	"f20": 0xffd1, // XK_F20

	// Numeric keypad keys
	"decimal":   0xffae, // XK_KP_Decimal
	"multiply":  0xffaa, // XK_KP_Multiply
	"add":       0xffab, // XK_KP_Add
	"divide":    0xffaf, // XK_KP_Divide
	"subtract":  0xffad, // XK_KP_Subtract
	"separator": 0xffac, // XK_KP_Separator

	// OpenAI-specific aliases[1], without adjusted for macOS keys
	//
	// [1]: https://github.com/openai/openai-cua-sample-app/blob/3751c8baa6376c0bbf6cceea2cdc0c0b42996e03/packages/runner-core/src/responses-loop.ts#L152-L210
	"ctrl":   0xffe3, // XK_Control_L
	"return": 0xff0d, // XK_Return
	"esc":    0xff1b, // XK_Escape
	"space":  0x0020, // XK_space
	"pgup":   0xff55, // XK_Page_Up
	"pgdn":   0xff56, // XK_Page_Down
	"up":     0xff52, // XK_Up
	"down":   0xff54, // XK_Down
	"left":   0xff51, // XK_Left
	"right":  0xff53, // XK_Right

	// Adjusted macOS keys
	//
	// For some reason in OpenAI's example these keys are swapped,
	// causing the commands on macOS to not work correctly.
	"cmd":     0xffe9, // XK_Meta_L
	"command": 0xffe9, // XK_Meta_L
	"option":  0xffe7, // XK_Alt_L
}
