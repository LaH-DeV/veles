package lexer

import (
	"fmt"
	"unicode"
	"unicode/utf8"
)

// Identifiers follow UAX #31 (D18 amended 2026-09-25): an identifier is
// `_` or an ID_Start character, then any number of ID_Continue characters.
// Before, every byte at or above 0x80 was an identifier byte, which let a
// no-break space, a zero-width space or a right-to-left override into a
// name — the "Trojan source" family (CVE-2021-42574), where a file reads
// one way to a person and compiles another.
//
// The properties are the standard's definitions, built from Go's unicode
// tables (which follow the Unicode version the Go release ships):
//
//	ID_Start    = L + Nl + Other_ID_Start − Pattern_Syntax − Pattern_White_Space
//	ID_Continue = ID_Start + Mn + Mc + Nd + Pc + Other_ID_Continue − (same)
//
// ASCII is decided without a table lookup; it is almost every byte a lexer
// sees. Identifiers are not NFC-normalized: two spellings of `é` are two
// names (the standard recommends normalizing; Go's standard library has no
// normalizer, and the cost is only a confusing "undeclared" error).

func isIDStart(r rune) bool {
	if r < utf8.RuneSelf {
		return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
	}
	if unicode.Is(unicode.Pattern_Syntax, r) || unicode.Is(unicode.Pattern_White_Space, r) {
		return false
	}
	return unicode.IsLetter(r) || unicode.Is(unicode.Nl, r) || unicode.Is(unicode.Other_ID_Start, r)
}

func isIDContinue(r rune) bool {
	if r < utf8.RuneSelf {
		return r == '_' || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
	}
	if isIDStart(r) {
		return true
	}
	if unicode.Is(unicode.Pattern_Syntax, r) || unicode.Is(unicode.Pattern_White_Space, r) {
		return false
	}
	return unicode.In(r, unicode.Mn, unicode.Mc, unicode.Nd, unicode.Pc, unicode.Other_ID_Continue)
}

// identStartLen is the byte length of an identifier-start character at
// src[i], or 0 when there is none.
func identStartLen(src string, i int) int {
	if i >= len(src) {
		return 0
	}
	if c := src[i]; c < utf8.RuneSelf {
		if isIDStart(rune(c)) {
			return 1
		}
		return 0
	}
	r, size := utf8.DecodeRuneInString(src[i:])
	if r == utf8.RuneError || !isIDStart(r) {
		return 0
	}
	return size
}

// identContinueLen is identStartLen for the characters after the first.
func identContinueLen(src string, i int) int {
	if i >= len(src) {
		return 0
	}
	if c := src[i]; c < utf8.RuneSelf {
		if isIDContinue(rune(c)) {
			return 1
		}
		return 0
	}
	r, size := utf8.DecodeRuneInString(src[i:])
	if r == utf8.RuneError || !isIDContinue(r) {
		return 0
	}
	return size
}

// identEnd is the end of the identifier that starts at i.
func identEnd(src string, i int) int {
	i += identStartLen(src, i)
	for {
		n := identContinueLen(src, i)
		if n == 0 {
			return i
		}
		i += n
	}
}

// isBidiControl reports the characters that reorder how text is displayed:
// the embeddings and overrides, the isolates, and the three marks. Inside
// code or a comment they are the Trojan-source attack; inside a string
// they are legitimate text, but belong in an escape where they can be seen.
func isBidiControl(r rune) bool {
	return (r >= 0x202A && r <= 0x202E) || (r >= 0x2066 && r <= 0x2069) || r == 0x200E || r == 0x200F || r == 0x061C
}

// describeRune spells an unexpected character for a diagnostic, saying
// what an invisible one is, since the reader cannot see it.
func describeRune(r rune) string {
	switch {
	case isBidiControl(r):
		return fmt.Sprintf("U+%04X, a bidirectional control character, which makes text display in a different order from how it compiles", r)
	case r == 0x00A0 || r == 0x202F || r == 0x2007:
		return fmt.Sprintf("U+%04X, a no-break space; use an ordinary space", r)
	case r == 0xFEFF:
		return "U+FEFF, a byte-order mark, which may only begin a file"
	case unicode.Is(unicode.Cf, r) || unicode.Is(unicode.Zs, r) || unicode.Is(unicode.Zl, r) || unicode.Is(unicode.Zp, r):
		return fmt.Sprintf("U+%04X, an invisible character", r)
	case showAsItself(r):
		return fmt.Sprintf("'%c' (U+%04X)", r, r)
	default:
		return fmt.Sprintf("U+%04X", r)
	}
}

// checkComment reports the bidirectional controls in a comment, src[start:end].
// In a comment one can make the code after it display as part of the
// comment, or the comment as code: the heart of the Trojan-source attack,
// which is why it is an error here and only a warning in a string.
func (lx *Lexer) checkComment(start, end int) {
	for i := start; i < end; {
		r, size := utf8.DecodeRuneInString(lx.src[i:end])
		if isBidiControl(r) {
			lx.errorf(i, i+size, "a comment contains %s; remove it", describeRune(r))
		}
		i += size
	}
}
