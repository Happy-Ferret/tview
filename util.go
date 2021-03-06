package tview

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/gdamore/tcell"
	runewidth "github.com/mattn/go-runewidth"
)

// Text alignment within a box.
const (
	AlignLeft = iota
	AlignCenter
	AlignRight
)

// Semigraphical runes.
const (
	GraphicsHoriBar             = '\u2500'
	GraphicsVertBar             = '\u2502'
	GraphicsTopLeftCorner       = '\u250c'
	GraphicsTopRightCorner      = '\u2510'
	GraphicsBottomLeftCorner    = '\u2514'
	GraphicsBottomRightCorner   = '\u2518'
	GraphicsLeftT               = '\u251c'
	GraphicsRightT              = '\u2524'
	GraphicsTopT                = '\u252c'
	GraphicsBottomT             = '\u2534'
	GraphicsCross               = '\u253c'
	GraphicsDbVertBar           = '\u2550'
	GraphicsDbHorBar            = '\u2551'
	GraphicsDbTopLeftCorner     = '\u2554'
	GraphicsDbTopRightCorner    = '\u2557'
	GraphicsDbBottomRightCorner = '\u255d'
	GraphicsDbBottomLeftCorner  = '\u255a'
	GraphicsEllipsis            = '\u2026'
)

// joints maps combinations of two graphical runes to the rune that results
// when joining the two in the same screen cell. The keys of this map are
// two-rune strings where the value of the first rune is lower than the value
// of the second rune. Identical runes are not contained.
var joints = map[string]rune{
	"\u2500\u2502": GraphicsCross,
	"\u2500\u250c": GraphicsTopT,
	"\u2500\u2510": GraphicsTopT,
	"\u2500\u2514": GraphicsBottomT,
	"\u2500\u2518": GraphicsBottomT,
	"\u2500\u251c": GraphicsCross,
	"\u2500\u2524": GraphicsCross,
	"\u2500\u252c": GraphicsTopT,
	"\u2500\u2534": GraphicsBottomT,
	"\u2500\u253c": GraphicsCross,
	"\u2502\u250c": GraphicsLeftT,
	"\u2502\u2510": GraphicsRightT,
	"\u2502\u2514": GraphicsLeftT,
	"\u2502\u2518": GraphicsRightT,
	"\u2502\u251c": GraphicsLeftT,
	"\u2502\u2524": GraphicsRightT,
	"\u2502\u252c": GraphicsCross,
	"\u2502\u2534": GraphicsCross,
	"\u2502\u253c": GraphicsCross,
	"\u250c\u2510": GraphicsTopT,
	"\u250c\u2514": GraphicsLeftT,
	"\u250c\u2518": GraphicsCross,
	"\u250c\u251c": GraphicsLeftT,
	"\u250c\u2524": GraphicsCross,
	"\u250c\u252c": GraphicsTopT,
	"\u250c\u2534": GraphicsCross,
	"\u250c\u253c": GraphicsCross,
	"\u2510\u2514": GraphicsCross,
	"\u2510\u2518": GraphicsRightT,
	"\u2510\u251c": GraphicsCross,
	"\u2510\u2524": GraphicsRightT,
	"\u2510\u252c": GraphicsTopT,
	"\u2510\u2534": GraphicsCross,
	"\u2510\u253c": GraphicsCross,
	"\u2514\u2518": GraphicsBottomT,
	"\u2514\u251c": GraphicsLeftT,
	"\u2514\u2524": GraphicsCross,
	"\u2514\u252c": GraphicsCross,
	"\u2514\u2534": GraphicsBottomT,
	"\u2514\u253c": GraphicsCross,
	"\u2518\u251c": GraphicsCross,
	"\u2518\u2524": GraphicsRightT,
	"\u2518\u252c": GraphicsCross,
	"\u2518\u2534": GraphicsBottomT,
	"\u2518\u253c": GraphicsCross,
	"\u251c\u2524": GraphicsCross,
	"\u251c\u252c": GraphicsCross,
	"\u251c\u2534": GraphicsCross,
	"\u251c\u253c": GraphicsCross,
	"\u2524\u252c": GraphicsCross,
	"\u2524\u2534": GraphicsCross,
	"\u2524\u253c": GraphicsCross,
	"\u252c\u2534": GraphicsCross,
	"\u252c\u253c": GraphicsCross,
	"\u2534\u253c": GraphicsCross,
}

// Common regular expressions.
var (
	colorPattern    = regexp.MustCompile(`\[([a-zA-Z]+|#[0-9a-zA-Z]{6})?(:([a-zA-Z]+|#[0-9a-zA-Z]{6})?(:([lbdru]+))?)?\]`)
	regionPattern   = regexp.MustCompile(`\["([a-zA-Z0-9_,;: \-\.]*)"\]`)
	escapePattern   = regexp.MustCompile(`\[([a-zA-Z0-9_,;: \-\."#]+)\[(\[*)\]`)
	boundaryPattern = regexp.MustCompile("([[:punct:]]\\s*|\\s+)")
	spacePattern    = regexp.MustCompile(`\s+`)
)

// Positions of substrings in regular expressions.
const (
	colorForegroundPos = 1
	colorBackgroundPos = 3
	colorFlagPos       = 5
)

// Predefined InputField acceptance functions.
var (
	// InputFieldInteger accepts integers.
	InputFieldInteger func(text string, ch rune) bool

	// InputFieldFloat accepts floating-point numbers.
	InputFieldFloat func(text string, ch rune) bool

	// InputFieldMaxLength returns an input field accept handler which accepts
	// input strings up to a given length. Use it like this:
	//
	//   inputField.SetAcceptanceFunc(InputFieldMaxLength(10)) // Accept up to 10 characters.
	InputFieldMaxLength func(maxLength int) func(text string, ch rune) bool
)

// Package initialization.
func init() {
	// Initialize the predefined input field handlers.
	InputFieldInteger = func(text string, ch rune) bool {
		if text == "-" {
			return true
		}
		_, err := strconv.Atoi(text)
		return err == nil
	}
	InputFieldFloat = func(text string, ch rune) bool {
		if text == "-" || text == "." || text == "-." {
			return true
		}
		_, err := strconv.ParseFloat(text, 64)
		return err == nil
	}
	InputFieldMaxLength = func(maxLength int) func(text string, ch rune) bool {
		return func(text string, ch rune) bool {
			return len([]rune(text)) <= maxLength
		}
	}
}

// styleFromTag takes the given style and modifies it based on the substrings
// extracted by the regular expression for color tags. The new style is returned
// as well as the flag indicating if any style attributes were explicitly
// specified (whose original value is also returned).
func styleFromTag(style tcell.Style, overwriteAttr bool, tagSubstrings []string) (tcell.Style, bool) {
	// Colors.
	if tagSubstrings[colorForegroundPos] != "" {
		color := tagSubstrings[colorForegroundPos]
		if color == "" {
			style = style.Foreground(tcell.ColorDefault)
		} else {
			style = style.Foreground(tcell.GetColor(color))
		}
	}
	if tagSubstrings[colorBackgroundPos-1] != "" {
		color := tagSubstrings[colorBackgroundPos]
		if color == "" {
			style = style.Background(tcell.ColorDefault)
		} else {
			style = style.Background(tcell.GetColor(color))
		}
	}

	// Flags.
	specified := tagSubstrings[colorFlagPos-1] != ""
	if specified {
		overwriteAttr = true
		style = style.Normal()
		for _, flag := range tagSubstrings[colorFlagPos] {
			switch flag {
			case 'l':
				style = style.Blink(true)
			case 'b':
				style = style.Bold(true)
			case 'd':
				style = style.Dim(true)
			case 'r':
				style = style.Reverse(true)
			case 'u':
				style = style.Underline(true)
			}
		}
	}
	return style, overwriteAttr
}

// overlayStyle mixes a bottom and a top style and returns the result. Top
// colors (other than tcell.ColorDefault) overwrite bottom colors. Top
// style attributes overwrite bottom style attributes only if overwriteAttr is
// true.
func overlayStyle(bottom, top tcell.Style, overwriteAttr bool) tcell.Style {
	style := bottom
	fg, bg, attr := top.Decompose()
	if bg != tcell.ColorDefault {
		style = style.Background(bg)
	}
	if fg != tcell.ColorDefault {
		style = style.Foreground(fg)
	}
	if overwriteAttr {
		style = style.Normal()
		style |= tcell.Style(attr)
	}
	return style
}

// decomposeString returns information about a string which may contain color
// tags. It returns the indices of the color tags (as returned by
// re.FindAllStringIndex()), the color tags themselves (as returned by
// re.FindAllStringSubmatch()), the indices of an escaped tags, the string
// stripped by any color tags and escaped, and the screen width of the stripped
// string.
func decomposeString(text string) (colorIndices [][]int, colors [][]string, escapeIndices [][]int, stripped string, width int) {
	// Get positions of color and escape tags.
	colorIndices = colorPattern.FindAllStringIndex(text, -1)
	colors = colorPattern.FindAllStringSubmatch(text, -1)
	escapeIndices = escapePattern.FindAllStringIndex(text, -1)

	// Because the color pattern detects empty tags, we need to filter them out.
	for i := len(colorIndices) - 1; i >= 0; i-- {
		if colorIndices[i][1]-colorIndices[i][0] == 2 {
			colorIndices = append(colorIndices[:i], colorIndices[i+1:]...)
			colors = append(colors[:i], colors[i+1:]...)
		}
	}

	// Remove the color tags from the original string.
	var from int
	buf := make([]byte, 0, len(text))
	for _, indices := range colorIndices {
		buf = append(buf, []byte(text[from:indices[0]])...)
		from = indices[1]
	}
	buf = append(buf, text[from:]...)

	// Escape string.
	stripped = string(escapePattern.ReplaceAll(buf, []byte("[$1$2]")))

	// Get the width of the stripped string.
	width = runewidth.StringWidth(stripped)

	return
}

// Print prints text onto the screen into the given box at (x,y,maxWidth,1),
// not exceeding that box. "align" is one of AlignLeft, AlignCenter, or
// AlignRight. The screen's background color will not be changed.
//
// You can change the colors and text styles mid-text by inserting a color tag.
// See the package description for details.
//
// Returns the number of actual runes printed (not including color tags) and the
// actual width used for the printed runes.
func Print(screen tcell.Screen, text string, x, y, maxWidth, align int, color tcell.Color) (int, int) {
	return printWithStyle(screen, text, x, y, maxWidth, align, tcell.StyleDefault.Foreground(color), false)
}

// printWithStyle works like Print() but it takes a style instead of just a
// foreground color. The overwriteAttr indicates whether or not a style's
// additional attributes (see tcell.AttrMask) should be overwritten.
func printWithStyle(screen tcell.Screen, text string, x, y, maxWidth, align int, style tcell.Style, overwriteAttr bool) (int, int) {
	if maxWidth < 0 {
		return 0, 0
	}

	// Decompose the text.
	colorIndices, colors, escapeIndices, strippedText, _ := decomposeString(text)

	// We deal with runes, not with bytes.
	runes := []rune(strippedText)

	// This helper function takes positions for a substring of "runes" and a start
	// style and returns the substring with the original tags and the new start
	// style.
	substring := func(from, to int, style tcell.Style, overwriteAttr bool) (string, tcell.Style, bool) {
		var colorPos, escapePos, runePos, startPos int
		for pos := range text {
			// Handle color tags.
			if colorPos < len(colorIndices) && pos >= colorIndices[colorPos][0] && pos < colorIndices[colorPos][1] {
				if pos == colorIndices[colorPos][1]-1 {
					if runePos <= from {
						style, overwriteAttr = styleFromTag(style, overwriteAttr, colors[colorPos])
					}
					colorPos++
				}
				continue
			}

			// Handle escape tags.
			if escapePos < len(escapeIndices) && pos >= escapeIndices[escapePos][0] && pos < escapeIndices[escapePos][1] {
				if pos == escapeIndices[escapePos][1]-1 {
					escapePos++
				} else if pos == escapeIndices[escapePos][1]-2 {
					continue
				}
			}

			// Check boundaries.
			if runePos == from {
				startPos = pos
			} else if runePos >= to {
				return text[startPos:pos], style, overwriteAttr
			}

			runePos++
		}

		return text[startPos:], style, overwriteAttr
	}

	// We want to reduce everything to AlignLeft.
	if align == AlignRight {
		width := 0
		start := len(runes)
		for index := start - 1; index >= 0; index-- {
			w := runewidth.RuneWidth(runes[index])
			if width+w > maxWidth {
				break
			}
			width += w
			start = index
		}
		text, style, overwriteAttr = substring(start, len(runes), style, overwriteAttr)
		return printWithStyle(screen, text, x+maxWidth-width, y, width, AlignLeft, style, overwriteAttr)
	} else if align == AlignCenter {
		width := runewidth.StringWidth(strippedText)
		if width == maxWidth {
			// Use the exact space.
			return printWithStyle(screen, text, x, y, maxWidth, AlignLeft, style, overwriteAttr)
		} else if width < maxWidth {
			// We have more space than we need.
			half := (maxWidth - width) / 2
			return printWithStyle(screen, text, x+half, y, maxWidth-half, AlignLeft, style, overwriteAttr)
		} else {
			// Chop off runes until we have a perfect fit.
			var choppedLeft, choppedRight, leftIndex, rightIndex int
			rightIndex = len(runes) - 1
			for rightIndex > leftIndex && width-choppedLeft-choppedRight > maxWidth {
				leftWidth := runewidth.RuneWidth(runes[leftIndex])
				rightWidth := runewidth.RuneWidth(runes[rightIndex])
				if choppedLeft < choppedRight {
					choppedLeft += leftWidth
					leftIndex++
				} else {
					choppedRight += rightWidth
					rightIndex--
				}
			}
			text, style, overwriteAttr = substring(leftIndex, rightIndex, style, overwriteAttr)
			return printWithStyle(screen, text, x, y, maxWidth, AlignLeft, style, overwriteAttr)
		}
	}

	// Draw text.
	drawn := 0
	drawnWidth := 0
	var colorPos, escapePos int
	for pos, ch := range text {
		// Handle color tags.
		if colorPos < len(colorIndices) && pos >= colorIndices[colorPos][0] && pos < colorIndices[colorPos][1] {
			if pos == colorIndices[colorPos][1]-1 {
				style, overwriteAttr = styleFromTag(style, overwriteAttr, colors[colorPos])
				colorPos++
			}
			continue
		}

		// Handle escape tags.
		if escapePos < len(escapeIndices) && pos >= escapeIndices[escapePos][0] && pos < escapeIndices[escapePos][1] {
			if pos == escapeIndices[escapePos][1]-1 {
				escapePos++
			} else if pos == escapeIndices[escapePos][1]-2 {
				continue
			}
		}

		// Check if we have enough space for this rune.
		chWidth := runewidth.RuneWidth(ch)
		if drawnWidth+chWidth > maxWidth {
			break
		}
		finalX := x + drawnWidth

		// Print the rune.
		_, _, finalStyle, _ := screen.GetContent(finalX, y)
		finalStyle = overlayStyle(finalStyle, style, overwriteAttr)
		for offset := 0; offset < chWidth; offset++ {
			// To avoid undesired effects, we place the same character in all cells.
			screen.SetContent(finalX+offset, y, ch, nil, finalStyle)
		}

		drawn++
		drawnWidth += chWidth
	}

	return drawn, drawnWidth
}

// PrintSimple prints white text to the screen at the given position.
func PrintSimple(screen tcell.Screen, text string, x, y int) {
	Print(screen, text, x, y, math.MaxInt32, AlignLeft, Styles.PrimaryTextColor)
}

// StringWidth returns the width of the given string needed to print it on
// screen. The text may contain color tags which are not counted.
func StringWidth(text string) int {
	_, _, _, _, width := decomposeString(text)
	return width
}

// WordWrap splits a text such that each resulting line does not exceed the
// given screen width. Possible split points are after any punctuation or
// whitespace. Whitespace after split points will be dropped.
//
// This function considers color tags to have no width.
//
// Text is always split at newline characters ('\n').
func WordWrap(text string, width int) (lines []string) {
	colorTagIndices, _, escapeIndices, strippedText, _ := decomposeString(text)

	// Find candidate breakpoints.
	breakPoints := boundaryPattern.FindAllStringIndex(strippedText, -1)

	// This helper function adds a new line to the result slice. The provided
	// positions are in stripped index space.
	addLine := func(from, to int) {
		// Shift indices back to original index space.
		var colorTagIndex, escapeIndex int
		for colorTagIndex < len(colorTagIndices) && to >= colorTagIndices[colorTagIndex][0] ||
			escapeIndex < len(escapeIndices) && to >= escapeIndices[escapeIndex][0] {
			past := 0
			if colorTagIndex < len(colorTagIndices) {
				tagWidth := colorTagIndices[colorTagIndex][1] - colorTagIndices[colorTagIndex][0]
				if colorTagIndices[colorTagIndex][0] < from {
					from += tagWidth
					to += tagWidth
					colorTagIndex++
				} else if colorTagIndices[colorTagIndex][0] < to {
					to += tagWidth
					colorTagIndex++
				} else {
					past++
				}
			} else {
				past++
			}
			if escapeIndex < len(escapeIndices) {
				tagWidth := escapeIndices[escapeIndex][1] - escapeIndices[escapeIndex][0]
				if escapeIndices[escapeIndex][0] < from {
					from += tagWidth
					to += tagWidth
					escapeIndex++
				} else if escapeIndices[escapeIndex][0] < to {
					to += tagWidth
					escapeIndex++
				} else {
					past++
				}
			} else {
				past++
			}
			if past == 2 {
				break // All other indices are beyond the requested string.
			}
		}
		lines = append(lines, text[from:to])
	}

	// Determine final breakpoints.
	var start, lastEnd, newStart, breakPoint int
	for {
		// What's our candidate string?
		var candidate string
		if breakPoint < len(breakPoints) {
			candidate = text[start:breakPoints[breakPoint][1]]
		} else {
			candidate = text[start:]
		}
		candidate = strings.TrimRightFunc(candidate, unicode.IsSpace)

		if runewidth.StringWidth(candidate) >= width {
			// We're past the available width.
			if lastEnd > start {
				// Use the previous candidate.
				addLine(start, lastEnd)
				start = newStart
			} else {
				// We have no previous candidate. Make a hard break.
				var lineWidth int
				for index, ch := range text {
					if index < start {
						continue
					}
					chWidth := runewidth.RuneWidth(ch)
					if lineWidth > 0 && lineWidth+chWidth >= width {
						addLine(start, index)
						start = index
						break
					}
					lineWidth += chWidth
				}
			}
		} else {
			// We haven't hit the right border yet.
			if breakPoint >= len(breakPoints) {
				// It's the last line. We're done.
				if len(candidate) > 0 {
					addLine(start, len(strippedText))
				}
				break
			} else {
				// We have a new candidate.
				lastEnd = start + len(candidate)
				newStart = breakPoints[breakPoint][1]
				breakPoint++
			}
		}
	}

	return
}

// PrintJoinedBorder prints a border graphics rune into the screen at the given
// position with the given color, joining it with any existing border graphics
// rune. Background colors are preserved. At this point, only regular single
// line borders are supported.
func PrintJoinedBorder(screen tcell.Screen, x, y int, ch rune, color tcell.Color) {
	previous, _, style, _ := screen.GetContent(x, y)
	style = style.Foreground(color)

	// What's the resulting rune?
	var result rune
	if ch == previous {
		result = ch
	} else {
		if ch < previous {
			previous, ch = ch, previous
		}
		result = joints[string(previous)+string(ch)]
	}
	if result == 0 {
		result = ch
	}

	// We only print something if we have something.
	screen.SetContent(x, y, result, nil, style)
}
