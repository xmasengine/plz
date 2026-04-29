
		JP_Imm16 start
		:hello
		LD_A_Imm8 'H'
		OUT_Port_A 7
		LD_A_Imm8 'E'
		OUT_Port_A 7
		LD_A_Imm8 'L'
		OUT_Port_A 7
		LD_A_Imm8 'L'
		OUT_Port_A 7
		LD_A_Imm8 'O'
		OUT_Port_A 7
		RET
		:start
		CALL_Imm16 hello
		HALT
