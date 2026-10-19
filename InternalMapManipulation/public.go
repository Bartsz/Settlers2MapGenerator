package internalmapmanipulation

// Snapshot is the immutable result used by API and CLI.

func Generate(payload []byte) (Snapshot, error) {
	req, err := decodeMapRequirements(payload)
	if err != nil {
		return Snapshot{}, err
	}
	return UnifiedGenerate(req)
}

func ValidateSWD(data []byte) error { _, err := parseSwd(data); return err }
