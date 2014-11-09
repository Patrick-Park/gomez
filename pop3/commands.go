package pop3

func cmdUSER(tx *transaction, param string) error {
	return nil
}

func cmdPASS(tx *transaction, param string) error {
	return nil
}

// RFC 1939 5. Transaction State STAT command
func cmdSTAT(tx *transaction, param string) error {
	return nil
}

// RFC 1939 5. Transaction State LIST command
func cmdLIST(tx *transaction, param string) error {
	return nil
}

// RFC 1939 5. Transaction State RETR command
func cmdRETR(tx *transaction, param string) error {
	return nil
}

// RFC 1939 5. Transaction State DELE command
func cmdDELE(tx *transaction, param string) error {
	return nil
}

// RFC 1939 5. Transaction State NOOP command
func cmdNOOP(tx *transaction, param string) error {
	return nil
}

// RFC 1939 5. Transaction State RSET command
func cmdRSET(tx *transaction, param string) error {
	return nil
}

// RFC 1939 5. Update State QUIT command
func cmdQUIT(tx *transaction, param string) error {
	return nil
}
