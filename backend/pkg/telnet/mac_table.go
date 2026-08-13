package telnet

import (
	"fmt"
	"strings"

	"github.com/ispcms/backend/internal/repositories"
)

// ParseRicherlinkMACTable parses the output of "show mac-address-table" from a
// Richerlink EPON OLT and returns only the entries that resolve to a specific ONU
// sub-interface (rows with an epon port AND an onu subif).
//
// CLI output format (space-delimited, variable column widths):
//
//	VLAN   MAC-ADDRESS     TYPE     PORT    SUBIF
//	----  --------------  -------  ------  -------
//	901   1c61.b462.ed3d  dynamic  epon1   onu12
//	1     1c18.4a67.afaa  dynamic  ge5
//
// Rows without a SUBIF column (GE uplink rows) are silently skipped.
// MAC addresses are converted from Cisco dotted-hex to colon-separated hex.
func ParseRicherlinkMACTable(output string) []repositories.MACTableEntry {
	var entries []repositories.MACTableEntry
	seen := map[string]bool{}

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		// Need: VLAN MAC TYPE PORT SUBIF (5 columns minimum)
		if len(fields) < 5 {
			continue
		}
		// Skip header / separator lines
		first := fields[0]
		if first == "VLAN" || strings.HasPrefix(first, "---") || strings.HasPrefix(first, "====") {
			continue
		}

		portStr := fields[3] // e.g. "epon1"
		subif := fields[4]   // e.g. "onu12"

		if !strings.HasPrefix(portStr, "epon") || !strings.HasPrefix(subif, "onu") {
			continue // GE uplink or unrecognised — skip
		}

		var portNum, onuSlot int
		if _, err := fmt.Sscanf(portStr[4:], "%d", &portNum); err != nil || portNum <= 0 {
			continue
		}
		if _, err := fmt.Sscanf(subif[3:], "%d", &onuSlot); err != nil {
			continue
		}

		mac := normalizeCiscoMAC(fields[1])
		key := fmt.Sprintf("%d:%d:%s", portNum, onuSlot, mac)
		if seen[key] {
			continue
		}
		seen[key] = true

		entries = append(entries, repositories.MACTableEntry{
			MAC:     mac,
			PortIdx: portNum,
			ONUSlot: onuSlot,
		})
	}
	return entries
}

// normalizeCiscoMAC converts a Cisco-style dotted-hex MAC "1c61.b462.ed3d"
// to colon-separated lowercase "1c:61:b4:62:ed:3d".
// The repositories.MACTableEntry normalizer handles colons fine.
func normalizeCiscoMAC(mac string) string {
	bare := strings.ToLower(strings.ReplaceAll(mac, ".", ""))
	if len(bare) != 12 {
		return mac // pass through; normalizeMACHex in repo will strip non-hex
	}
	return fmt.Sprintf("%s:%s:%s:%s:%s:%s",
		bare[0:2], bare[2:4], bare[4:6], bare[6:8], bare[8:10], bare[10:12])
}
