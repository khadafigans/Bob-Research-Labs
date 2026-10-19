package agent

import (
	"log"
	"path/filepath"

	"github.com/Autumn-27/norma/agentcore"
	"github.com/Autumn-27/norma/noaadapter"
)

// noaWarn returns a diagnostics sink tagging non-fatal noa messages with the
// session, routed through the package logger (agents have no per-instance one).
func noaWarn(session string) func(string) {
	return func(msg string) { log.Printf("[noa] %s: %s", session, msg) }
}

// 。 compaction :noaadapter.Enable ,
// (Compactor)、Compress , Enable 
// ( compaction )。 agent  noaEnabledFn , run 

// enableNoa  noa  opts。archiveRoot 
// ( workDir, agent  <workDir>/noa ,/),sessionID
//
//  opts.Compaction, agentcore 「」。
func enableNoa(opts *agentcore.Options, enabled func() bool, archiveRoot, sessionID string, onWarn func(string)) {
	if enabled == nil || !enabled() {
		return
	}
	if opts.OnWarn == nil {
		opts.OnWarn = onWarn
	}
	if err := noaadapter.Enable(opts, noaadapter.Options{
		ArchiveBaseDir: filepath.Join(archiveRoot, "noa"),
		SessionID:      sessionID,
		OnWarn:         onWarn,
	}); err != nil {
		if onWarn != nil {
			onWarn("noa ,:" + err.Error())
		}
		return
	}
	// Compactor  Compaction, agentcore ;。
	opts.Compaction = nil
}
