package term

import (
	"bytes"
	"fmt"
	"strings"
)

var frameBuffer = new(bytes.Buffer)

var HomePos = Point{X: 1, Y: 1}

func SetPosAnsiCode(x int, y int) string {
	return fmt.Sprintf("\x1B[%d;%dH", y, x)
}

func SetPos(point Point) {
	frameBuffer.WriteString(SetPosAnsiCode(point.X, point.Y))
}

func MoveLinesAnsiCode(n int) string {
	var suffix = "B"
	if n < 0 {
		suffix = "A"
	}
	return fmt.Sprintf("\x1B[%d%s", n, suffix)
}

func MoveLines(n int) {
	frameBuffer.WriteString(MoveLinesAnsiCode(n))
}

const ClearScreenAnsiCode = "\x1b[2J"

func ClearScreen() {
	frameBuffer.WriteString(ClearScreenAnsiCode)
}

func RenderStyledBlock(rect Rect, sb StyledBlock) {
	lines := sb.Lines
	if len(lines) > rect.Height {
		lines = lines[:rect.Height]
	}
	point := rect.Point
	for _, line := range lines {
		SetPos(point)
		frameBuffer.WriteString(ansiReset)
		remainingWidth := rect.Width
		for _, span := range line.Spans {
			trim := TrimStringToWidth(span.Text, remainingWidth)
			frameBuffer.WriteString(AnsiCode(span.Style))
			frameBuffer.WriteString(trim.Trimmed)
			remainingWidth -= trim.Width
			if len(trim.Tail) > 0 || remainingWidth == 0 {
				break
			}
		}

		frameBuffer.WriteString(ansiReset)

		// erase what's behind
		for i := 0; i < remainingWidth; i++ {
			frameBuffer.WriteString(" ")
		}
		point.Y++
	}
	frameBuffer.WriteString(ansiReset)
}

func RenderRawText(rect Rect, text string) {
	lines := strings.Split(text, "\n")
	if len(lines) > rect.Height {
		lines = lines[:rect.Height]
	}

	point := rect.Point
	frameBuffer.WriteString(ansiReset)
	for _, line := range lines {
		SetPos(point)
		segments := SplitRawToSegments(line)
		remainingWidth := rect.Width
		for _, span := range segments {
			frameBuffer.WriteString(span.Control)
			if remainingWidth > 0 {
				trim := TrimStringToWidth(span.Text, remainingWidth)
				frameBuffer.WriteString(trim.Trimmed)
				remainingWidth -= trim.Width
				// we don't break; we just stop printing text but continue to print control chars
				// in case there's some space but we can't fit the remainig text in it, set remainigng width to 0 so we don't print any more text
				if len(trim.Tail) > 0 {
					remainingWidth = 0
				}
			}
		}
		point.Y++
	}
	frameBuffer.WriteString(ansiReset)
}

func AnsiCode(span Style) string {
	var buf = new(strings.Builder)
	buf.WriteString("\x1b[0")

	for i := AttrFirst; i <= AttrLast; i++ {
		if span.Attr&(1<<i) != 0 {
			fmt.Fprintf(buf, ";%d", i+1)
		}
	}

	// if we are not blocking foreground color setting
	if span.Attr&DefaultForeground == 0 {
		if span.Foreground < 8 {
			fmt.Fprintf(buf, ";3%d", span.Foreground)
		} else {
			fmt.Fprintf(buf, ";38;5;%d", span.Foreground)
		}
	}

	// if we are not blocking background color setting
	if span.Attr&DefaultBackground == 0 {
		if span.Background < 8 {
			fmt.Fprintf(buf, ";4%d", span.Background)
		} else {
			fmt.Fprintf(buf, ";48;5;%d", span.Background)
		}

	}

	buf.WriteString("m")

	return buf.String()
}
