package keymap

const (
	XK_Shift_L = 0xffe1
)

var KeyMap = map[string]rune{
	// Packer compatibility using X11 header[1] as a reference
	//
	// [1]: https://gitlab.freedesktop.org/xorg/proto/xorgproto/-/blob/master/include/X11/keysymdef.h

	// Editing keys
	"bs":       0xff08, // XK_BackSpace
	"del":      0xffff, // XK_Delete
	"enter":    0xff0d, // XK_Return
	"return":   0xff0d, // XK_Return
	"esc":      0xff1b, // XK_Escape
	"tab":      0xff09, // XK_Tab
	"spacebar": 0x0020, // XK_space

	// Navigation keys
	"insert":   0xff63, // XK_Insert
	"home":     0xff50, // XK_Home
	"end":      0xff57, // XK_End
	"pageUp":   0xff55, // XK_Prior
	"pageDown": 0xff56, // XK_Next

	// Arrow keys
	"up":    0xff52, // XK_Up
	"down":  0xff54, // XK_Down
	"left":  0xff51, // XK_Left
	"right": 0xff53, // XK_Right

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

	// System / app keys
	"menu": 0xff67, // XK_Menu

	// Modifier keys
	"leftAlt":    0xffe9,     // XK_Alt_L
	"rightAlt":   0xffea,     // XK_Alt_R
	"leftCtrl":   0xffe3,     // XK_Control_L
	"rightCtrl":  0xffe4,     // XK_Control_R
	"leftShift":  XK_Shift_L, // XK_Shift_L
	"rightShift": 0xffe2,     // XK_Shift_R
	"leftSuper":  0xffeb,     // XK_Super_L
	"rightSuper": 0xffec,     // XK_Super_R

	// Packer builder for Tart VMs compatibility
	//
	// These were introduced in https://github.com/cirruslabs/packer-plugin-tart/pull/218.

	// Modifier keys
	"leftCommand":  0xffe9, // Alias for XK_Alt_L
	"rightCommand": 0xffea, // Alias for XK_Alt_R
	"leftOption":   0xffe7, // XK_Meta_L
	"rightOption":  0xffe8, // XK_Meta_R
}
