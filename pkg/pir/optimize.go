package pir

// Optimize runs peephole optimisations on a PIR program.
//
// It walks the instruction stream tracking constant values on the virtual
// data stack. When a binary or unary operation has known-constant operands,
// the result is computed at compile time (constant folding). It also
// eliminates identities (x+0 → x, x*1 → x, x*0 → 0, x&0 → 0,
// x&allOnes → x, x|0 → x, x|allOnes → allOnes, x^0 → x,
// x/1 → x, x%1 → 0) and reduces multiplication/division/modulo
// by powers of two to shift/and operations.
//
// Note: x − x → 0 is handled only when both operands are known
// constants (constant folding). The compiler never emits DUP
// followed by SUB, so the DUP + SUB pattern is absent from
// real programs.
//
// The implementation replaces superseded instructions with NOP and compacts
// them away in a final pass.
func Optimize(prog *Program) *Program {
	o := &optimizer{prog: make([]Instr, len(prog.Instrs))}
	copy(o.prog, prog.Instrs)
	o.run()
	o.compact()
	return &Program{Instrs: o.prog}
}

type optVal struct {
	known bool
	val   uint16
	idx   int // instruction index that produced this value
}

type optimizer struct {
	prog  []Instr
	stack []optVal
}

func (o *optimizer) run() {
	for i := 0; i < len(o.prog); i++ {
		instr := o.prog[i]
		switch instr.Op {
		case PUSH_B:
			o.push(optVal{known: true, val: uint16(instr.Operand.Num) & 0xFF, idx: i})
		case PUSH_W:
			o.push(optVal{known: true, val: instr.Operand.Num, idx: i})

		case ADD_B, ADD_W:
			o.foldBinop(i, func(a, b uint16) uint16 { return a + b },
				func(a, b uint16) bool { return b == 0 })
		case SUB_B, SUB_W:
			o.foldBinop(i, func(a, b uint16) uint16 { return a - b },
				func(a, b uint16) bool { return b == 0 })
		case MUL_B, MUL_W:
			o.foldMul(i, instr.Op)
		case DIV_B, DIV_W:
			o.foldDiv(i, instr.Op)
		case MOD_B, MOD_W:
			o.foldMod(i, instr.Op)

		case AND_B, AND_W:
			o.foldAnd(i, instr.Op)
		case OR_B, OR_W:
			o.foldOr(i, instr.Op)
		case XOR_B, XOR_W:
			o.foldBinop(i, func(a, b uint16) uint16 { return a ^ b },
				func(a, b uint16) bool { return b == 0 })
		case SHL_B, SHL_W:
			o.foldBinop(i, func(a, b uint16) uint16 { return a << (b & 0x0F) },
				func(a, b uint16) bool { return b == 0 })
		case SHR_B, SHR_W:
			o.foldBinop(i, func(a, b uint16) uint16 { return a >> (b & 0x0F) },
				func(a, b uint16) bool { return b == 0 })

		case NEG_B:
			o.foldUnop(i, func(a uint16) uint16 { return uint16((-a) & 0xFF) })
		case NEG_W:
			o.foldUnop(i, func(a uint16) uint16 { return uint16(-int32(a)) })
		case NOT_B:
			o.foldUnop(i, func(a uint16) uint16 {
				if a&0xFF == 0 { return 1 }; return 0
			})
		case NOT_W:
			o.foldUnop(i, func(a uint16) uint16 {
				if a == 0 { return 1 }; return 0
			})
		case CAST_B:
			o.foldUnop(i, func(a uint16) uint16 { return a & 0xFF })
		case CAST_W:
			o.foldUnop(i, func(a uint16) uint16 { return a })

		case DUP:
			if len(o.stack) > 0 {
				t := o.stack[len(o.stack)-1]
				o.push(optVal{known: t.known, val: t.val, idx: i})
			} else {
				o.push(optVal{idx: i})
			}
		case DROP:
			o.pop()
		case SWAP:
			if len(o.stack) >= 2 {
				o.stack[len(o.stack)-1], o.stack[len(o.stack)-2] =
					o.stack[len(o.stack)-2], o.stack[len(o.stack)-1]
			}

		case GET_B, GET_W, PUSH_A, PUSH_D, IN_B, IN_W, SEED:
			o.push(optVal{idx: i})
		case PUT_B, PUT_W, OUT_B, OUT_W, SWITCH:
			o.pop()
		case READ_B, READ_W:
			o.pop()
			o.push(optVal{idx: i})
		case WRITE_B, WRITE_W:
			o.pop()
			o.pop()
		case SAVE, LOAD:
			o.pop()
			o.pop()
			o.pop()
		case SLEEP:
			o.pop()

		case IS_B, IS_W:
			o.foldIS(i)

		// Stack-preserving instructions (no data stack effect)
		case NOP, TAG, ROUTE, JOB, BYE, YIELD, HLT,
			ALLOC, VAR, PRIORITY, AT, BANK, INT, NMI,
			ENI, DII, FRAME, LOCAL_B, LOCAL_W,
			STOP, START, SRAM_ON, SRAM_OFF, PRAGMA, INLINE,
			DATA_B, DATA_W, DATA_STR:
			// no stack effect — preserve known constant tracking

		// Unconditional control flow — clear stack
		case GO, DONE, DONE_INTERRUPT, DONE_NMI:
			o.stack = o.stack[:0]

		// Conditional jump — pops condition value
		case GO_IF:
			o.pop()

		default:
			o.unknown()
		}
	}
}

// ── Folding ──

func (o *optimizer) foldBinop(i int, fold func(a, b uint16) uint16, identity func(a, b uint16) bool) {
	if len(o.stack) < 2 {
		o.push(optVal{idx: i})
		return
	}
	r := o.pop()
	l := o.pop()

	if l.known && r.known {
		result := fold(l.val, r.val)
		o.nopAt(l.idx)
		o.nopAt(r.idx)
		o.replace(i, PUSH_W, result)
		o.push(optVal{known: true, val: result, idx: i})
		return
	}

	if identity != nil && r.known && identity(l.val, r.val) {
		o.nopAt(r.idx)
		o.nop(i)
		o.push(l)
		return
	}

	o.push(optVal{idx: i})
}

func (o *optimizer) foldMul(i int, op Instruction) {
	if len(o.stack) < 2 {
		o.push(optVal{idx: i})
		return
	}
	r := o.pop()
	l := o.pop()

	if l.known && r.known {
		result := l.val * r.val
		o.nopAt(l.idx)
		o.nopAt(r.idx)
		o.replace(i, PUSH_W, result)
		o.push(optVal{known: true, val: result, idx: i})
		return
	}

	if r.known {
		switch {
		case r.val == 0:
			o.nopAt(l.idx)
			o.nopAt(r.idx)
			o.replace(i, PUSH_W, 0)
			o.push(optVal{known: true, val: 0, idx: i})
			return
		case r.val == 1:
			o.nopAt(r.idx)
			o.nop(i)
			o.push(l)
			return
		case isPow2(r.val):
			shift := log2(r.val)
			o.nopAt(r.idx)
			shOp := SHL_W
			if op == MUL_B {
				shOp = SHL_B
			}
			o.replace(i, PUSH_B, uint16(shift))
			o.insertAfter(i, Instr{Op: shOp})
			o.push(optVal{idx: i + 1})
			return
		}
	}

	o.push(optVal{idx: i})
}

func (o *optimizer) foldDiv(i int, op Instruction) {
	if len(o.stack) < 2 {
		o.push(optVal{idx: i})
		return
	}
	r := o.pop()
	l := o.pop()

	if l.known && r.known {
		if r.val == 0 {
			o.push(optVal{idx: i})
			return
		}
		result := l.val / r.val
		o.nopAt(l.idx)
		o.nopAt(r.idx)
		o.replace(i, PUSH_W, result)
		o.push(optVal{known: true, val: result, idx: i})
		return
	}

	if r.known {
		switch {
		case r.val == 1:
			o.nopAt(r.idx)
			o.nop(i)
			o.push(l)
			return
		case isPow2(r.val):
			shift := log2(r.val)
			o.nopAt(r.idx)
			srOp := SHR_W
			if op == DIV_B {
				srOp = SHR_B
			}
			o.replace(i, PUSH_B, uint16(shift))
			o.insertAfter(i, Instr{Op: srOp})
			o.push(optVal{idx: i + 1})
			return
		}
	}

	o.push(optVal{idx: i})
}

func (o *optimizer) foldMod(i int, op Instruction) {
	if len(o.stack) < 2 {
		o.push(optVal{idx: i})
		return
	}
	r := o.pop()
	l := o.pop()

	if l.known && r.known {
		if r.val == 0 {
			o.push(optVal{idx: i})
			return
		}
		result := l.val % r.val
		o.nopAt(l.idx)
		o.nopAt(r.idx)
		o.replace(i, PUSH_W, result)
		o.push(optVal{known: true, val: result, idx: i})
		return
	}

	if r.known {
		switch {
		case r.val == 1:
			o.nopAt(l.idx)
			o.nopAt(r.idx)
			o.replace(i, PUSH_W, 0)
			o.push(optVal{known: true, val: 0, idx: i})
			return
		case isPow2(r.val):
			mask := r.val - 1
			o.nopAt(r.idx)
			andOp := AND_W
			if op == MOD_B {
				andOp = AND_B
			}
			o.replace(i, PUSH_W, mask)
			o.insertAfter(i, Instr{Op: andOp})
			o.push(optVal{idx: i + 1})
			return
		}
	}

	o.push(optVal{idx: i})
}

func (o *optimizer) foldUnop(i int, fold func(uint16) uint16) {
	if len(o.stack) == 0 {
		o.push(optVal{idx: i})
		return
	}
	v := o.pop()
	if v.known {
		result := fold(v.val)
		o.nopAt(v.idx)
		o.replace(i, PUSH_W, result)
		o.push(optVal{known: true, val: result, idx: i})
		return
	}
	o.push(optVal{idx: i})
}

func (o *optimizer) foldAnd(i int, op Instruction) {
	if len(o.stack) < 2 {
		o.push(optVal{idx: i})
		return
	}
	r := o.pop()
	l := o.pop()

	if l.known && r.known {
		result := l.val & r.val
		o.nopAt(l.idx)
		o.nopAt(r.idx)
		o.replace(i, PUSH_W, result)
		o.push(optVal{known: true, val: result, idx: i})
		return
	}

	allOnes := uint16(0xFFFF)
	if op == AND_B {
		allOnes = 0xFF
	}

	if r.known {
		switch {
		case r.val == 0:
			o.nopAt(l.idx)
			o.nopAt(r.idx)
			o.replace(i, PUSH_W, 0)
			o.push(optVal{known: true, val: 0, idx: i})
			return
		case r.val == allOnes:
			o.nopAt(r.idx)
			o.nop(i)
			o.push(l)
			return
		}
	}

	o.push(optVal{idx: i})
}

func (o *optimizer) foldOr(i int, op Instruction) {
	if len(o.stack) < 2 {
		o.push(optVal{idx: i})
		return
	}
	r := o.pop()
	l := o.pop()

	if l.known && r.known {
		result := l.val | r.val
		o.nopAt(l.idx)
		o.nopAt(r.idx)
		o.replace(i, PUSH_W, result)
		o.push(optVal{known: true, val: result, idx: i})
		return
	}

	allOnes := uint16(0xFFFF)
	if op == OR_B {
		allOnes = 0xFF
	}

	if r.known {
		switch {
		case r.val == 0:
			o.nopAt(r.idx)
			o.nop(i)
			o.push(l)
			return
		case r.val == allOnes:
			o.nopAt(l.idx)
			o.nopAt(r.idx)
			o.replace(i, PUSH_W, allOnes)
			o.push(optVal{known: true, val: allOnes, idx: i})
			return
		}
	}

	o.push(optVal{idx: i})
}

func (o *optimizer) foldIS(i int) {
	if len(o.stack) < 2 {
		o.push(optVal{idx: i})
		return
	}
	r := o.pop() // TOS = right operand
	l := o.pop() // NEXT = left operand

	if l.known && r.known {
		var result bool
		switch o.prog[i].Operand.Cond {
		case CondLT:
			result = l.val < r.val
		case CondGT:
			result = l.val > r.val
		case CondLE:
			result = l.val <= r.val
		case CondGE:
			result = l.val >= r.val
		case CondEQ:
			result = l.val == r.val
		case CondNE:
			result = l.val != r.val
		}
		val := uint16(0)
		if result {
			val = 1
		}
		o.nopAt(l.idx)
		o.nopAt(r.idx)
		o.replace(i, PUSH_W, val)
		o.push(optVal{known: true, val: val, idx: i})
		return
	}
	o.push(optVal{idx: i})
}

// ── Stack helpers ──

func (o *optimizer) push(v optVal) {
	o.stack = append(o.stack, v)
}

func (o *optimizer) pop() optVal {
	if len(o.stack) == 0 {
		return optVal{idx: -1}
	}
	v := o.stack[len(o.stack)-1]
	o.stack = o.stack[:len(o.stack)-1]
	return v
}

func (o *optimizer) unknown() {
	for i := range o.stack {
		o.stack[i].known = false
	}
}

// ── Instruction replacement ──

func (o *optimizer) nop(i int) {
	o.prog[i].Op = NOP
}

func (o *optimizer) nopAt(idx int) {
	if idx >= 0 && idx < len(o.prog) {
		o.prog[idx].Op = NOP
	}
}

func (o *optimizer) replace(i int, op Instruction, val uint16) {
	o.prog[i] = Instr{Op: op, Operand: Operand{Type: OpNumber, Num: val}}
}

func (o *optimizer) insertAfter(i int, instr Instr) {
	o.prog = append(o.prog, Instr{})       // make room
	copy(o.prog[i+2:], o.prog[i+1:])       // shift right
	o.prog[i+1] = instr
	for j := range o.stack {
		if o.stack[j].idx > i {
			o.stack[j].idx++
		}
	}
}

func (o *optimizer) compact() {
	dst := 0
	for _, instr := range o.prog {
		if instr.Op != NOP {
			o.prog[dst] = instr
			dst++
		}
	}
	o.prog = o.prog[:dst]
}

// ── Helpers ──

func isPow2(n uint16) bool {
	return n > 0 && n&(n-1) == 0
}

func log2(n uint16) uint16 {
	v := uint16(0)
	for n > 1 {
		n >>= 1
		v++
	}
	return v
}
