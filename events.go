package term

import (
	"bytes"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
	"unicode/utf8"

	"go.hasen.dev/generic"
	"golang.org/x/sys/unix"
	"golang.org/x/term"
)

/*
// TEMP for debugging
func init() {
	vbeam.InitRotatingLogger("term")
}
*/

var state *term.State

var stdout = os.Stdout

func Deinit() {
	term.Restore(0, state)
}

func Init() {
	state, _ = term.MakeRaw(0)
}

func HideCursor() {
	cmd("\x1b[?25l")
}

func ShowCursor() {
	cmd("\x1b[?25h")
}

func EnableMouse() {
	cmd("\x1b[?1003;1006h")
}

func DisableMouse() {
	cmd("\x1b[?1003;1006l")
}

func EnterAltScreen() {
	cmd("\x1b[?1049h")
}

func ExitAltScreen() {
	cmd("\033[?1049l")
}

func cmd(cmd string) {
	stdout.WriteString(cmd)
}

type EventType byte

const (
	KeyboardEvent EventType = iota
	MouseEvent
)

func (et EventType) String() string {
	switch et {
	case KeyboardEvent:
		return "keyboard"
	case MouseEvent:
		return "mouse"
	default:
		return "---"
	}
}

type MouseAction byte

const (
	MousePress MouseAction = iota

	// NOTE: We only get one mouse release event even if multiple buttons are pressed
	// The good news is: no further events are sent until all buttons are released
	// So it's safe to treat a mouse release event as if all buttons are released
	MouseRelease

	MouseMotion

	WheelUp
	WheelDown
)

func (ms MouseAction) String() string { // for debugging
	switch ms {
	case MousePress:
		return "press"
	case MouseRelease:
		return "release"
	case MouseMotion:
		return "motion"
	case WheelUp:
		return "wheel_up"
	case WheelDown:
		return "wheel_down"
	default:
		return "---"
	}
}

func (mb MouseButton) String() string {
	switch mb {
	case MousePrimary:
		return "left"
	case MouseSecondary:
		return "right"
	case MouseMiddle:
		return "middle"
	case MouseButtonNone:
		return "no_button"
	default:
		return "---"
	}
}

type MouseButton byte

const (
	MousePrimary MouseButton = iota
	MouseMiddle
	MouseSecondary
	MouseButtonNone
)

type Event struct {
	Raw  []byte
	Time time.Time

	Type EventType

	// Keyboard
	Key rune

	// Mouse Event Properties
	MouseAction   MouseAction
	MousePos      Point
	MouseButton   MouseButton
	MouseModShift bool
	MouseModCtrl  bool
	MouseModAlt   bool
}

var TermSize Size

func StartEventLoop(frameFn func(event []Event), fps int) {
	var bufLock sync.Mutex      // mutually exclusive lock for `buf`
	var buf = new(bytes.Buffer) // collects input data as it arrives (TODO: make ring buffer?)

	// Read goroutine
	go func() {
		var sigio = make(chan os.Signal, 10)
		signal.Notify(sigio, syscall.SIGIO)

		// setup non-blocking input with sigio notification
		unix.SetNonblock(0, true)
		unix.FcntlInt(0, unix.F_SETOWN, unix.Getpid())
		unix.FcntlInt(0, unix.F_SETFL, unix.O_ASYNC)

		chunk := make([]byte, 1<<12) // 4k
		for range sigio {
			n, _ := os.Stdin.Read(chunk)
			if n > 0 {
				bufLock.Lock()
				buf.Write(chunk[:n])
				bufLock.Unlock()
			}
		}
	}()

	/*
		We are using stdout to control the UI on the screen.
		Any code that prints to stdout is going to cause screen corruption.

		We also want to avoid excessive writing to stdout, so we have what
		amounts to a "frame buffer", and if there's no change to it from
		last frame, we should not need to flush.

		If someone writes to Stdout or Stderr, that causes two problems

		- They compete with us for what the screen should look like, breaking
		  all the meticulous work we did to control the layout of text and
		  colors on the screen

		- The text they print could contain valuable information to the user,
		  but the user will not really be able to see it, because _we_ are
		  taking total control over the screen.

		To mitigate both points, we do the following:

		- "Subvert" the os provided Stdout and Stderr variables to point to
		  actual files on the file system. If the user needs to see the output
		  from code, they can `tail` that file.

		- Check the modtime on stdout and stderr. If the user manages to write
		  to stdout without referencing os.Stdout, we will know because the
		  modtime on stdout changed, and we'll flush the framebuffer even if
		  it has no changes since last frame.
	*/

	{
		// subvert stdout and stderr to local files
		fout, _ := os.Create(".stdout")
		ferr, _ := os.Create(".stderr")
		os.Stdout = fout
		os.Stderr = ferr
		// go func() {
		// 	ticker := time.NewTicker(time.Second * 1)
		// 	for range ticker.C {
		// 		ferr.Sync()
		// 		fout.Sync()
		// 	}
		// }()
		// generic.AddExitCleanup(func() {
		// 	ferr.Sync()
		// 	fout.Sync()
		// })
	}

	getMod := func(f *os.File) time.Time {
		s, _ := f.Stat()
		return s.ModTime()
	}

	const ForceFlushInterval = 1 * time.Second

	var prevFrameBuffer []byte
	var prevMod1 time.Time
	var prevMod2 time.Time
	var prevFlush time.Time
	// Event loop
	ticker := time.NewTicker(time.Millisecond * time.Duration(1000/fps))
	for range ticker.C {
		// update terminal size
		// FIXME: might be expensive to call at every frame! find a cheaper way
		TermSize.Width, TermSize.Height, _ = term.GetSize(0)

		// parse input buffer for events!
		bufLock.Lock()
		events := consumeInputEvents(buf)
		bufLock.Unlock()

		frameBuffer.Reset()
		ClearScreen()
		SetPos(Point{1, 1})

		frameFn(events)
		// check if we need to render:
		// - is the frame buffer different from last time?
		// - did someone else write to stdout and stderr?
		mod1 := getMod(os.Stdout)
		mod2 := getMod(os.Stderr)

		now := time.Now()

		currFrameBuffer := frameBuffer.Bytes()
		if !bytes.Equal(prevFrameBuffer, currFrameBuffer) ||
			mod1 != prevMod1 || mod2 != prevMod2 ||
			now.Sub(prevFlush) > ForceFlushInterval {
			stdout.Write(currFrameBuffer)
			prevFlush = now
		}
		// get the mod again because we just wrote to stdout
		prevMod1 = getMod(os.Stdout)
		prevMod2 = getMod(os.Stderr)

		// NOTE: there's a potential race condition, where if someone writes to
		// stdout or stderr in the time between us flushing the framebuffer and
		// reading the timestamp for stdout and stderr, we would not know about
		// the screen being corrupted. That's why we use a ForceFlushInterval

		prevFrameBuffer = generic.Clone(currFrameBuffer)
	}
}

// consumeInputEvents parses events in the event data buffer one by one
// and returns a list of parsed events. If the last event data is incomplete,
// it will be put back into the byte buffer.
func consumeInputEvents(buf *bytes.Buffer) (list []Event) {
	if buf.Len() == 0 {
		return
	}
	// clone data as we will pass it to ParseEvent which will keep
	// segments of it for reference
	data := generic.Clone(buf.Bytes())

	// log.Printf("size: %d, data: %x", len(data), data)

	// We will put back the unconsumed parts of the events after we consume
	// all available events
	buf.Reset()

	initialSize := len(data)

	for len(data) > 0 {
		event, size := ParseEvent(data)
		if size == 0 {
			break
		} else {
			data = data[size:]
			list = append(list, event)
		}

		// special case: large input stream, last character is esc
		// it could be the start of an escape sequence that the rest
		// of is still waiting in the next frame or something
		//
		// This is not just a theoretical issue; we've seen it in practice
		if initialSize > 12 && len(data) == 1 && data[0] == 27 {
			break
		}
	}

	// data is now the remaining data that could not be parsed as a complete
	// event.
	buf.Write(data)

	return
}

// ParseEvent parses the next event in the input byte stream Returns the event
// data and how many bytes were consumed.
//
// If the byte stream does not start with a complete event, size 0 is returned
// and the returned event must be discarded as invalid
//
// NOTE: ParseEvent assumes it can keep references to segments of data. If data
// is volatile, make a copy of it before sending it here.

func ParseEvent(data []byte) (ev Event, size int) {
	defer func() {
		if err := recover(); err != nil {
			// FIXME this is going to get subverted; do we care?
			fmt.Printf("\r\nRecovered from error %v\r\n", err)
		}
	}()

	ev.Time = time.Now()

	if bytes.HasPrefix(data, ControlSequenceStartBytes) {
		segmentSize, isValid := FindEndOfControlSequence(generic.UnsafeString(data))
		if !isValid {
			size = 0
			return
		}
		size = segmentSize
		ctrlSeq := data[:segmentSize]
		ctrlSeq = bytes.TrimPrefix(ctrlSeq, ControlSequenceStartBytes)
		ev.Raw = ctrlSeq
		last := ctrlSeq[len(ctrlSeq)-1]
		first := ctrlSeq[0]
		var isMouseEvent = first == '<' && (last == 'M' || last == 'm')
		if isMouseEvent {
			ev.Type = MouseEvent
			mouseData := string(ctrlSeq[1 : len(ctrlSeq)-1])
			parts := strings.Split(mouseData, ";")
			if len(parts) == 3 {
				buttonCode, _ := strconv.Atoi(parts[0])
				ev.MousePos.X, _ = strconv.Atoi(parts[1])
				ev.MousePos.Y, _ = strconv.Atoi(parts[2])

				// The button code is a bit compliated
				// if the event is press or release, the buttonCode is (0, 1, 2)
				// and the press/release is indicated by last char being M or m
				// If it's a motion event, 32 is added to the code
				// modifier button states are indicated by bit flags

				const ShiftFlag = 1 << 2
				const AltFlag = 1 << 3
				const CtrlFlag = 1 << 4

				const MotionFlag = 1 << 5
				const WheelFlag = 1 << 6

				ev.MouseModShift = buttonCode&ShiftFlag != 0
				ev.MouseModAlt = buttonCode&AltFlag != 0
				ev.MouseModCtrl = buttonCode&CtrlFlag != 0

				if buttonCode&WheelFlag != 0 {
					ev.MouseButton = MouseButtonNone
					ev.MouseAction = WheelUp
					if buttonCode&1 != 0 {
						ev.MouseAction = WheelDown
					}
				} else if buttonCode&MotionFlag != 0 {
					ev.MouseAction = MouseMotion
					ev.MouseButton = MouseButton((buttonCode - 32) & 3)
				} else {
					ev.MouseButton = MouseButton(buttonCode & 3)
					switch last {
					case 'm':
						ev.MouseAction = MouseRelease
					case 'M':
						ev.MouseAction = MousePress
					}
				}
			}
		}
	} else { // Not a control sequence; must be a key event (rune)
		ev.Type = KeyboardEvent
		ev.Key, size = utf8.DecodeRune(data)
		ev.Raw = data[:size]
	}

	return
}
