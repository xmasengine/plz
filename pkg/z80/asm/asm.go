// asm is a simple two pass assembler with macros for the z80,
// which supports several formats.
package asm

import "io"
import "strconv"
import "text/scanner"

import "github.com/xmasengine/plz/pkg/z80/isa"

func AssembleBinary(rd io.Reader) []isa.Opcode {
	scan := (&scanner.Scanner{}).Init(rd)
	res := []isa.Opcode{}
	labels := map[string]int{}
	references := map[uint16]string{}
	defLabel := false
	for token := scan.Scan(); token != scanner.EOF; token = scan.Scan() {
		// First very simple and silly, we ignore all comments and whitespaces.
		switch token {
		case scanner.Ident:
			{
				ident := scan.TokenText()
				instruction := false
				for i := int(isa.NOP); i <= int(isa.RST_0x38); i++ {
					o := isa.Opcode(i)
					if o.String() == ident {
						res = append(res, o)
						instruction = true
						break
					}
				}
				if !instruction {
					if defLabel {
						defLabel = false
						labels[ident] = len(res)
					} else {
						references[uint16(len(res))] = ident
						res = append(res, isa.Opcode(0), isa.Opcode(0))
					}
				}
			}
		case ':':
			defLabel = true
		case scanner.Int:
			{
				nums := scan.TokenText()
				number, _ := strconv.Atoi(nums)
				if number < 0 {
					res = append(res, isa.Opcode(int8(number)))
				} else if number > 255 {
					u16 := uint16(number)
					lo := isa.Opcode(uint8(u16 & 255))
					hi := isa.Opcode(uint8(u16 >> 8))
					res = append(res, lo, hi)
				} else {
					res = append(res, isa.Opcode(uint8(number)))
				}
			}

		case scanner.Char:
			{
				chars := scan.TokenText()
				if len(chars) > 2 {
					char, _, _, err := strconv.UnquoteChar(chars[1:len(chars)-1], '\'')
					if err != nil {
						println("error: ", err.Error())
					}
					res = append(res, isa.Opcode(char))
				}
			}
		}
	}
	for k, v := range labels {
		println("label", k, v)
	}
	for at, v := range references {
		println("ref", at, v)
		ptr, ok := labels[v]
		if ok {
			println("update reference", v, at, ptr)
			res[at] = isa.Opcode(ptr & 255)
			res[at+1] = isa.Opcode(ptr >> 8)
		} else {
			println("error: undefined reference to ", v)
		}
	}
	return res
}
