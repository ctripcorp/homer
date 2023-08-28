package internal

%%{
	machine decoder;
	write data;
}%%

func ParseCSeq(data []byte) (c []byte) {
	cs, p, pe := 0, 0, len(data)
	mark := 0

	%%{
		ALPHA = 0x41..0x5a | 0x61..0x7a;
		DIGIT = 0x30..0x39;

		CRLF = "\r\n";
		SP = " ";
		HTAB = "\t";
		WSP = SP | HTAB;
		LWS = ( WSP* CRLF )? WSP+;
		SWS = LWS?;
		HCOLON = ( SP | HTAB )* ":" SWS;

		INVITEm = 0x49.0x4e.0x56.0x49.0x54.0x45;
		ACKm = 0x41.0x43.0x4b;
		OPTIONSm = 0x4f.0x50.0x54.0x49.0x4f.0x4e.0x53;
		BYEm = 0x42.0x59.0x45;
		CANCELm = 0x43.0x41.0x4e.0x43.0x45.0x4c;
		REGISTERm = 0x52.0x45.0x47.0x49.0x53.0x54.0x45.0x52;
		INFOm = 0x49.0x4e.0x46.0x4f;
		PRACKm = 0x50.0x52.0x41.0x43.0x4b;
		SUBSCRIBEm = 0x53.0x55.0x42.0x53.0x43.0x52.0x49.0x42.0x45;
		NOTIFYm = 0x4e.0x4f.0x54.0x49.0x46.0x59;
		UPDATEm = 0x55.0x50.0x44.0x41.0x54.0x45;
		MESSAGEm = 0x4d.0x45.0x53.0x53.0x41.0x47.0x45;
		REFERm = 0x52.0x45.0x46.0x45.0x52;
		PUBLISHm = 0x50.0x55.0x42.0x4c.0x49.0x53.0x48;
		KDMQm = 0x4b.0x44.0x4d.0x51;
		Method = INVITEm | ACKm | OPTIONSm | BYEm | CANCELm | REGISTERm | INFOm | PRACKm | SUBSCRIBEm | NOTIFYm | UPDATEm | MESSAGEm | REFERm | PUBLISHm | KDMQm;

		action mark {
			mark = p
		}

		action parseMethod{
			c=data[mark:p]
		}
	
		CSeq = any* "CSeq"i HCOLON DIGIT+ LWS Method >mark %parseMethod;
		main := CSeq :>CRLF @{ return c } ;

		write init;
		write exec;
	}%%
	return c
}