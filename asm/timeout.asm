		JP_Imm16 start
		:loop
		LD_A_Imm8 '+' OUT_Port_A 7
		:start
		JP_Imm16 loop
		HALT

