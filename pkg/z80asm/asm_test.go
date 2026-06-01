package z80asm

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

type ffs map[string]string

func (f ffs) open(filename string) (io.ReadCloser, error) {
	contents, ok := f[filename]
	if !ok {
		return nil, os.ErrNotExist
	}
	return ioutil.NopCloser(strings.NewReader(contents)), nil
}

func toHex(bs []byte) string {
	var r []string
	for _, b := range bs {
		r = append(r, fmt.Sprintf("%02x", b))
	}
	return strings.Join(r, " ")
}

func testFailureSnippet(t *testing.T, nextCore Z80Core, fs ffs, mustContain string) {
	desc := fs["a.asm"]
	asm, err := NewAssembler(UseNextCore(nextCore))
	if err != nil {
		t.Fatalf("%q: failed to create assembler: %v", desc, err)
	}
	asm.opener = fs.open
	err = asm.AssembleFile("a.asm")
	if err == nil {
		t.Errorf("%q: assembler succeeded, expected match %q", desc, mustContain)
		return
	}
	if !strings.Contains(err.Error(), mustContain) {
		t.Errorf("%q: failure %q does not match %q", desc, err.Error(), mustContain)
	}
}

func testSnippet(t *testing.T, nextCore Z80Core, org int, fs ffs, want []byte) {
	desc := fs["a.asm"]
	asm, err := NewAssembler(UseNextCore(nextCore))
	if err != nil {
		t.Fatalf("%q: failed to create assembler: %v", desc, err)
	}
	asm.opener = fs.open
	if err := asm.AssembleFile("a.asm"); err != nil {
		t.Errorf("%q: assembler with nextCore=%d produced error: %v", desc, nextCore, err)
		return
	}
	ram := asm.RAM()
	for i := 0; i < 65536; i++ {
		if i < org || i >= org+len(want) {
			if ram[i] != 0 {
				t.Errorf("%q: byte %04x = %02x, want 0", desc, i, ram[i])
			}
		}
	}
	if got := ram[org : org+len(want)]; !reflect.DeepEqual(got, want) {
		t.Errorf("%q: assembled with nextCore=%d at %04x\ngot:\n%s\nwant:\n%s", desc, nextCore, org, toHex(got), toHex(want))
	}
}

func TestAsmSnippets(t *testing.T) {
	testcases := []struct {
		fs       ffs
		nextCore Z80Core
		want     []byte
	}{
		{
			fs: ffs{
				"a.asm": "xor a",
			},
			want: b(0xaf),
		},
		{
			fs: ffs{
				"a.asm": "ld bc, 42",
			},
			want: b(0x01, 42, 0),
		},
		{
			fs: ffs{
				"a.asm": "ld hl, 0x4243",
			},
			want: b(0x21, 0x43, 0x42),
		},
		{
			fs: ffs{
				"a.asm": "ld hl, 0x4243; ld bc, 0x1023",
			},
			want: b(0x21, 0x43, 0x42, 0x01, 0x23, 0x10),
		},
		{
			fs: ffs{
				"a.asm": "ld a, 0; ld b, 3; ld h, a; ld l, -2",
			},
			want: b(0x3e, 0, 0x06, 3, 0x67, 0x2e, 254),
		},
		{
			fs: ffs{
				"a.asm": ".label ld hl, label",
			},
			want: b(0x21, 0x00, 0x80),
		},
		{
			fs: ffs{
				"a.asm": "label: ld hl, label",
			},
			want: b(0x21, 0x00, 0x80),
		},
		{
			fs: ffs{
				"a.asm": "label: .xx ld hl, xx; label2: .xx ld hl, xx",
			},
			want: b(0x21, 0x00, 0x80, 0x21, 0x03, 0x80),
		},
		{
			fs: ffs{
				"a.asm": ".label push bc; jr label",
			},
			want: b(0xc5, 0x18, 0xfd),
		},
		{
			fs: ffs{
				"a.asm": "rst 0x20",
			},
			want: b(0xe7),
		},
		{
			fs: ffs{
				"a.asm": `db 1, 2, 3, 'h', '\n', '\t', 42`,
			},
			want: b(1, 2, 3, uint8('h'), uint8('\n'), uint8('\t'), 42),
		},
		{
			fs: ffs{
				"a.asm": `dw 1, 2, 256`,
			},
			want: b(1, 0, 2, 0, 0, 1),
		},
		{
			fs: ffs{
				"a.asm": `ds "hello\n"`,
			},
			want: []byte("hello\n"),
		},
		{
			fs: ffs{
				"a.asm": "ds `hello\\n`",
			},
			want: []byte("hello\\n"),
		},
		{
			fs: ffs{
				"a.asm": `rrca ; ret ; di`,
			},
			want: b(0x0f, 0xc9, 0xf3),
		},
		{
			fs: ffs{
				"a.asm": `ld a, (hl)`,
			},
			want: b(0x7e),
		},
		{
			fs: ffs{
				"a.asm": "ld a, (data); .data db 1, 2",
			},
			want: b(0x3a, 0x03, 0x80, 1, 2),
		},
		{
			fs: ffs{
				"a.asm": "out (42), a; out (c), h; in a, (10); in b, (c)",
			},
			want: b(0xd3, 42, 0xed, 0x61, 0xdb, 10, 0xed, 0x40),
		},
		{
			fs: ffs{
				"a.asm": ".loop jr nz, loop",
			},
			want: b(0x20, 0xfe),
		},
		{
			fs: ffs{
				"a.asm": "jr forwards ; db 42; .forwards ret",
			},
			want: b(0x18, 0x01, 42, 0xc9),
		},
		{
			fs: ffs{
				"a.asm": "\n\n\n\n/* Hello */\n\n\n",
			},
			want: []byte{},
		},
		{
			fs: ffs{
				"a.asm": "add ix, bc; ld ix, (1049); ld h, (ix+5); ld h, (ix-1)",
			},
			want: []byte{0xdd, 0x09, 0xdd, 0x2a, 25, 4, 0xdd, 0x66, 5, 0xdd, 0x66, 256 - 1},
		},
		{
			fs: ffs{
				"a.asm": "bit 4, (ix+10); set 0, (ix-9); res 1, (ix+0)",
			},
			want: []byte{0xdd, 0xcb, 10, 0x66, 0xdd, 0xcb, 256 - 9, 0xc6, 0xdd, 0xcb, 0, 0x8e},
		},
		{
			fs: ffs{
				"a.asm": "bit 4, (iy+10); set 0, (iy-9); res 1, (iy+0)",
			},
			want: []byte{0xfd, 0xcb, 10, 0x66, 0xfd, 0xcb, 256 - 9, 0xc6, 0xfd, 0xcb, 0, 0x8e},
		},
		{
			// We should be able to write (ix) and (iy) instead of (ix+0) and (iy+0).
			fs: ffs{
				"a.asm": "res 1, (ix); res 1, (iy)",
			},
			want: []byte{0xdd, 0xcb, 0, 0x8e, 0xfd, 0xcb, 0, 0x8e},
		},

		{
			nextCore: Z80CoreNext1,
			fs: ffs{
				"a.asm": "ldix; ldws; ldirx; lddx; lddrx; ldpirx",
			},
			want: []byte{0xed, 0xa4, 0xed, 0xa5, 0xed, 0xb4, 0xed, 0xac, 0xed, 0xbc, 0xed, 0xb7},
		},
		{
			nextCore: Z80CoreNext1,
			fs: ffs{
				"a.asm": "outinb; mul d, e; add hl, a; add de, a; add bc, a",
			},
			want: []byte{0xed, 0x90, 0xed, 0x30, 0xed, 0x31, 0xed, 0x32, 0xed, 0x33},
		},
		{
			nextCore: Z80CoreNext1,
			fs: ffs{
				"a.asm": "add hl, 0xa823; add de, 0x0102; add bc, 0x6543",
			},
			want: []byte{0xed, 0x34, 0x23, 0xa8, 0xed, 0x35, 0x02, 0x01, 0xed, 0x36, 0x43, 0x65},
		},
		{
			nextCore: Z80CoreNext1,
			fs: ffs{
				"a.asm": "swapnib; mirror a; pixeldn; pixelad; setae",
			},
			want: []byte{0xed, 0x23, 0xed, 0x24, 0xed, 0x93, 0xed, 0x94, 0xed, 0x95},
		},
		{
			nextCore: Z80CoreNext1,
			fs: ffs{
				"a.asm": "push 0xabcd; test 0x5a",
			},
			want: []byte{0xed, 0x8a, 0xab, 0xcd, 0xed, 0x27, 0x5a},
		},
		{
			nextCore: Z80CoreNext1,
			fs: ffs{
				"a.asm": "nextreg 0xab, 0x42; nextreg 0xfa, a",
			},
			want: []byte{0xed, 0x91, 0xab, 0x42, 0xed, 0x92, 0xfa},
		},
		{
			nextCore: Z80CoreNext2,
			fs: ffs{
				"a.asm": "bsla de, b; bsra de, b; bsrl de, b; bsrf de, b; brlc de, b; jp (c)",
			},
			want: []byte{0xed, 0x28, 0xed, 0x29, 0xed, 0x2a, 0xed, 0x2b, 0xed, 0x2c, 0xed, 0x98},
		},
		{
			// Test relocation: we set pc to 0x1000 but compile at 0x8000.
			fs: ffs{
				"a.asm": "org 0x1000, 0x8000; db 0xff; .label; dw label",
			},
			want: []byte{0xff, 0x01, 0x10},
		},

		{
			fs: ffs{
				"a.asm": "const x = 0xabcd; dw x & 0xf7f",
			},
			want: []byte{0x4d, 0x0b},
		},
		{
			// test we can define a const that depends on a later label!
			fs: ffs{
				"a.asm": "org 0x8000; const x = label + 1; dw x; .label",
			},
			want: []byte{0x03, 0x80},
		},
		{
			fs: ffs{
				"a.asm": `db 0x01; const x = 0xa0; include "b.asm"; db 0x42`,
				"b.asm": `db 0x02, x`,
			},
			want: []byte{0x01, 0x02, 0xa0, 0x42},
		},
	}
	for _, tc := range testcases {
		for c := Z80Core(0); c < 3; c++ {
			if c >= tc.nextCore {
				testSnippet(t, c, 0x8000, tc.fs, tc.want)
			} else {
				testFailureSnippet(t, c, tc.fs, "")
			}
		}
	}
}

func testMultipleErrors(t *testing.T, desc, src string, wantCount int) {
	fs := ffs{"a.asm": src}
	asm, err := NewAssembler()
	if err != nil {
		t.Fatalf("%q: failed to create assembler: %v", desc, err)
	}
	asm.opener = fs.open
	err = asm.AssembleFile("a.asm")
	if err == nil {
		t.Errorf("%q: assembler succeeded, expected many errors", desc)
		return
	}
	lines := strings.Split(err.Error(), "\n")
	if len(lines) != wantCount {
		t.Errorf("%q: expected %d errors, got %d: %v", desc, wantCount, len(lines), err.Error())
		return
	}
	for _, line := range lines {
		if !strings.Contains(line, "a.asm") {
			t.Errorf("%q: error line %q does not contain filename", desc, line)
		}
	}

}

func TestParseManyErrors(t *testing.T) {
	testCases := []struct {
		desc, src string
		wantCount int
	}{
		{
			desc: "lots of errors",
			src: `
				ld hl, 12(
				xor b, c
				jp backwards
				ld bc, a ; db 256
			`,
			wantCount: 5,
		},
		{
			desc: "just one error!",
			src: `
				ld hl, )1+2+3
				ld bc, 42
			`,
			wantCount: 1,
		},
		{
			desc:      "just one error, one line",
			src:       "ld hl, )1+2+3 ; ld bc, 42",
			wantCount: 1,
		},
		{
			desc:      "one line, two errors",
			src:       "ld hl, )1+2+3 ; ld bc, (a)",
			wantCount: 2,
		},
		{
			desc:      "two lines, two errors",
			src:       "ld hl, )1+2+3\nld bc, (a)",
			wantCount: 2,
		},
		{
			desc:      "one line, two errors",
			src:       "ld bc, (a);ld bc, (a)",
			wantCount: 2,
		},
		{
			desc:      "one line, two errors",
			src:       "ld hl, 1+2);ld bc, (a)",
			wantCount: 2,
		},
		{
			desc:      "type error, two errors",
			src:       "ld hl, \"fred\";ld bc, (a)",
			wantCount: 2,
		},
		{
			desc:      "two many commas, times two",
			src:       "ld hl, 1, a; ld bc, hl, de",
			wantCount: 2,
		},
		{
			desc:      "ix/iy+n out of range",
			src:       "ld a, (ix+128); ld a, (ix-129); ld a, (iy+128) ; ld a, (iy-129)",
			wantCount: 4,
		},
	}
	for _, tc := range testCases {
		testMultipleErrors(t, tc.desc, tc.src, tc.wantCount)
	}
}

func TestParseErrors(t *testing.T) {
	testCases := []struct {
		asm     string
		wantErr string // partial match
	}{
		{"xor a, b", "no suitable"},
		{"ld hl, (42", ")"},
		{"ld a, (1+2*3", ")"},
		{"ld a, )1+2*3", "unexpected token \")\""},
		{"ld a, 2+3+", "EOF"},
		{"ld a, 1 ld b, 2", "unexpected identifier \"ld\""},
		{"ld b, (123)", "no suitable"},
		{"xor a,", "unexpected trailing ,"},
		{"xor missing", "label"},
		{"ld hl, 6/(4-4)", "zero"},
		{"ld hl, 6%(4-4)", "zero"},
		{"db 256", "not in the range"},
		{"dw 65536", "not in the range"},
		{"label: ld hl, 42 ; label: ld bc, 42", "label \"label\" redefined"},
		{"a: .label ld hl, 42 ; .label: ld bc, 42", "label \"a.label\" redefined"},
		{"ld z, (1+2)", "(1 + 2)"},
		{"ld z, 1+(2*3)", "1 + 2 * 3"},
		{"ld z, 1*(2+3)", "1 * (2 + 3)"},
		{"ld z, 1+2+3", "1 + 2 + 3"},
		{"ld z, 1+(2+3)", "1 + (2 + 3)"},
		{"ld z, (1+2)+3", "1 + 2 + 3"},
		{"ld a, x; const x = 42", "use of const \"x\" before defin"},
		{`db 0x42; include "a.asm"`, "recursive"},
	}
	for _, tc := range testCases {
		testFailureSnippet(t, 0, ffs{"a.asm": tc.asm}, tc.wantErr)
	}
}

// TestErrorLocationFormat verifies all assembly errors include source location
// (filename:line.column), a key benefit of the error-returning refactor.
func TestErrorLocationFormat(t *testing.T) {
	testCases := []struct {
		desc    string
		asm     string
		wantErr string // substring match
	}{
		{"unknown command", "foobar", "a.asm:1."},
		{"out of range db", "db 256", "a.asm:1."},
		{"out of range dw", "dw 65536", "a.asm:1."},
		{"bad expr", "ld hl, 1+", "a.asm:1."},
		{"label redefined", "x: nop; x: nop", "a.asm:1."},
		{"divide by zero", "ld hl, 6/(4-4)", "a.asm:1."},
		{"const before def", "ld a, x; const x = 42", "a.asm:1."},
		{"no suitable variant", "xor a, b", "a.asm:1."},
		{"recursive include", `db 0x42; include "a.asm"`, "a.asm:1."},
		{"bad org zero args", "org", "a.asm:1."},
		{"bad org too many", "org 1, 2, 3", "a.asm:1."},
		{"bad const syntax", "const x", "a.asm:1."},
		{"const redefinition", "const x = 1; const x = 2", "a.asm:1."},
	}

	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			testFailureSnippet(t, 0, ffs{"a.asm": tc.asm}, tc.wantErr)
		})
	}
}

func testAssembles(t *testing.T, asm string) {
	t.Helper()
	a, err := NewAssembler()
	if err != nil {
		t.Fatalf("NewAssembler failed: %v", err)
	}
	a.opener = ffs{"a.asm": asm}.open
	if err := a.AssembleFile("a.asm"); err != nil {
		t.Errorf("expected success, got error: %v", err)
	}
}

// TestExpressionErrors verifies that edge cases in expression evaluation
// produce clean error messages instead of crashes.
func TestExpressionErrors(t *testing.T) {
	testCases := []struct {
		desc    string
		asm     string
		wantErr string // empty means expected to succeed
	}{
		{"negative shift left", "ld hl, 1<<-1", "shift must be positive"},
		{"negative shift right", "ld hl, 1>>-1", "shift must be positive"},
		{"shift by zero", "org 0x6000; .label ld hl, 1>>0", ""},
		{"mod by one", "org 0x6000; .label ld hl, 7%1", ""},
		{"complex and-shift", "org 0x6000; .label ld hl, ((16>>4)&1)", ""},
		{"logical or short-circuit", "org 0x6000; .label ld hl, 1||(2/0)", ""},
		{"logical and short-circuit", "org 0x6000; .label ld hl, 0&&(2/0)", ""},
		{"nested unary", "org 0x6000; .label ld hl, -(-5)", ""},
		{"double negation", "org 0x6000; .label ld hl, !!42", ""},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			if tc.wantErr == "" {
				testAssembles(t, tc.asm)
			} else {
				testFailureSnippet(t, 0, ffs{"a.asm": tc.asm}, tc.wantErr)
			}
		})
	}
}

// TestOrgEdgeCases tests the org directive with invalid inputs.
func TestOrgEdgeCases(t *testing.T) {
	testCases := []struct {
		desc    string
		asm     string
		wantErr string
	}{
		{"negative", "org -1", "out of range"},
		{"overflow", "org 65536", "out of range"},
	}
	for _, tc := range testCases {
		t.Run(tc.desc, func(t *testing.T) {
			testFailureSnippet(t, 0, ffs{"a.asm": tc.asm}, tc.wantErr)
		})
	}
}

func TestIntExpressions(t *testing.T) {
	testCases := []struct {
		expr string // An arithmetic operation.
		want uint16
	}{
		{"1+2", 3},
		{"7*4", 28},
		{"1-2", 65536 - 1},
		{"8/4", 2},
		{"-1+2", 1},
		{"1+2*3", 7},
		{"2*3+4", 10},
		{"2*(3+4)", 14},
		{"(1+2)*3", 9},
		{"8*8*8", 512},
		{"label+1", 0x6001},
		{"label/2", 0x3000},
		{"label*2", 0xc000},
		{"label+1*2", 0x6002},
		{"label*2+1", 0xc001},
		{"label-1", 0x5fff},
		{"label+label", 0xc000},
		{"16>>2", 4},
		{"2<<3", 16},
		{"1==2", 0},
		{"1+2==4-1", 1},
		{"1!=2", 1},
		{"1+2!=4-1", 0},
		{"10+20<10+30", 1},
		{"10+20<10+20", 0},
		{"10+20<=10+20", 1},
		{"10+20>10+30", 0},
		{"10+20>10+20", 0},
		{"10+20>=10+20", 1},
		{"10+30>10+20", 1},
		{"(16>>4)&1", 1},
		{"(16>>4)&3", 1},
		{"(7>>1)&2", 2},
		{"5|12", 13},
		{"5&12", 4},
		{"7%3", 1},
		{"^10", 65536 - 11},
		{"^0", 65535},
		{"^0+1", 0},
		{"!(1==0)", 1},
		{"!(0==0)", 0},
		{"-(1+2)", 65536 - 3},
		{"7&^2", 5},
		{"42||badness", 42},
		{"42&&43", 43},
		{"42||43", 42},
		{"0&&badness", 0},
		{"1&&0||3", 3},
		{"1&&2||3", 2},
		{"1||(2/0)", 1},
		{"0&&2||3", 3},
		{"1==2 && 2==2", 0},
		{"1==2 || 2==2", 1},
		{"1==2 || !(2==2)", 0},
		{"3-2-1", 0},
		{"8/4*2", 4},
	}
	for _, tc := range testCases {
		fs := ffs{
			"a.asm": fmt.Sprintf("org 0x6000 ; .label ld hl, %s", tc.expr),
		}
		want := b(0x21, byte(tc.want%256), byte(tc.want/256))
		testSnippet(t, 0, 0x6000, fs, want)
	}
}

func TestBank(t *testing.T) {
	fs := ffs{
		"a.asm": "bank 1\ndb 0xaa, 0xbb",
	}
	asm, err := NewAssembler()
	if err != nil {
		t.Fatalf("NewAssembler: %v", err)
	}
	asm.opener = fs.open
	if err := asm.AssembleFile("a.asm"); err != nil {
		t.Fatalf("assemble: %v", err)
	}
	ram := asm.RAM()
	if ram[0x8000] != 0xaa {
		t.Errorf("ram[0x8000] = %02x, want aa", ram[0x8000])
	}
	if ram[0x8001] != 0xbb {
		t.Errorf("ram[0x8001] = %02x, want bb", ram[0x8001])
	}
}

func TestBankSequential(t *testing.T) {
	fs := ffs{
		"a.asm": "bank 0\ndb 0x01\nbank 1\ndb 0x02\nbank 2\ndb 0x03",
	}
	asm, err := NewAssembler()
	if err != nil {
		t.Fatalf("NewAssembler: %v", err)
	}
	asm.opener = fs.open
	if err := asm.AssembleFile("a.asm"); err != nil {
		t.Fatalf("assemble: %v", err)
	}
	ram := asm.RAM()
	if ram[0x4000] != 0x01 {
		t.Errorf("bank 0 (0x4000) = %02x, want 01", ram[0x4000])
	}
	if ram[0x8000] != 0x02 {
		t.Errorf("bank 1 (0x8000) = %02x, want 02", ram[0x8000])
	}
	if ram[0xC000] != 0x03 {
		t.Errorf("bank 2 (0xC000) = %02x, want 03", ram[0xC000])
	}
}

func TestBankThenOrg(t *testing.T) {
	fs := ffs{
		"a.asm": "bank 1\ndb 0x01\norg 0x1000\ndb 0x02",
	}
	asm, err := NewAssembler()
	if err != nil {
		t.Fatalf("NewAssembler: %v", err)
	}
	asm.opener = fs.open
	if err := asm.AssembleFile("a.asm"); err != nil {
		t.Fatalf("assemble: %v", err)
	}
	ram := asm.RAM()
	if ram[0x8000] != 0x01 {
		t.Errorf("bank 1 (0x8000) = %02x, want 01", ram[0x8000])
	}
	if ram[0x1000] != 0x02 {
		t.Errorf("org 0x1000 = %02x, want 02", ram[0x1000])
	}
}

func TestBankSize(t *testing.T) {
	// Override bank size to 8KB, write to bank 1.
	fs := ffs{
		"a.asm": "banksize 0x2000\nbank 1\ndb 0xaa",
	}
	asm, err := NewAssembler()
	if err != nil {
		t.Fatalf("NewAssembler: %v", err)
	}
	asm.opener = fs.open
	if err := asm.AssembleFile("a.asm"); err != nil {
		t.Fatalf("assemble: %v", err)
	}
	ram := asm.RAM()
	// bankAt=0x4000, bankSize=0x2000 → bank 1 target = 0x4000 + 1*0x2000 = 0x6000
	if ram[0x6000] != 0xaa {
		t.Errorf("bank 1 with 8KB pages (0x6000) = %02x, want aa", ram[0x6000])
	}
}

func TestBankAt(t *testing.T) {
	// Override bank window to 0x8000, use default 16KB size.
	fs := ffs{
		"a.asm": "bankat 0x8000\nbank 1\ndb 0xbb",
	}
	asm, err := NewAssembler()
	if err != nil {
		t.Fatalf("NewAssembler: %v", err)
	}
	asm.opener = fs.open
	if err := asm.AssembleFile("a.asm"); err != nil {
		t.Fatalf("assemble: %v", err)
	}
	ram := asm.RAM()
	// bankAt=0x8000, bankSize=0x4000 → bank 1 target = 0x8000 + 1*0x4000 = 0xC000
	if ram[0xC000] != 0xbb {
		t.Errorf("bank 1 at 0x8000 window (0xC000) = %02x, want bb", ram[0xC000])
	}
}

func TestBankCustomConfig(t *testing.T) {
	// Custom bank size (32KB) and window (0x1000).
	fs := ffs{
		"a.asm": "banksize 0x8000\nbankat 0x1000\nbank 2\ndb 0xcc",
	}
	asm, err := NewAssembler()
	if err != nil {
		t.Fatalf("NewAssembler: %v", err)
	}
	asm.opener = fs.open
	if err := asm.AssembleFile("a.asm"); err != nil {
		t.Fatalf("assemble: %v", err)
	}
	ram := asm.RAM()
	// bankAt=0x1000, bankSize=0x8000 → bank 2 target = 0x1000 + 2*0x8000 = 0x11000
	if ram[0x11000] != 0xcc {
		t.Errorf("bank 2 custom config (0x11000) = %02x, want cc", ram[0x11000])
	}
}
