// Package conformance runs an adapter binary against the CLI protocol and verifies protocol
// behavior, contract schema validity, rawHash correctness, and resultHash determinism
// (ARCHITECTURE.md Section 4). Passing conformance is the merge bar for community adapters.
package conformance

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"

	"github.com/djmagro/outlays/core/internal/contract"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var fiscalYearRe = regexp.MustCompile(`^\d{4}(-\d{2})?$`)

var manifestRequired = []string{
	"adapterId", "jurisdiction", "datasets", "adapterVersion",
	"contractVersion", "license", "maintainer",
}

// Volatile fact fields excluded from the resultHash recomputation. MUST match the SDKs.
var volatileFactFields = []string{"factId", "runId", "insertedAt"}

// Result is the outcome of a conformance run.
type Result struct {
	Checks     []Check
	ResultHash string
}

// Check is one named assertion with pass/fail and detail.
type Check struct {
	Name   string
	Pass   bool
	Detail string
}

func (r *Result) add(name string, pass bool, detail string) {
	r.Checks = append(r.Checks, Check{Name: name, Pass: pass, Detail: detail})
}

// Passed reports whether every check passed.
func (r *Result) Passed() bool {
	for _, c := range r.Checks {
		if !c.Pass {
			return false
		}
	}
	return true
}

// Run executes the adapter (given as a command and its leading args, e.g. {"node","/p/cli.js"})
// through the full protocol and returns a Result. workDir holds the two fetch runs.
func Run(adapterCmd []string, year, workDir string) (*Result, error) {
	if len(adapterCmd) == 0 {
		return nil, fmt.Errorf("empty adapter command")
	}
	res := &Result{}

	// --- info ---
	infoOut, _, err := runCmd(adapterCmd, "info")
	if err != nil {
		res.add("info: exit 0", false, err.Error())
		return res, nil
	}
	res.add("info: exit 0", true, "")
	var manifest map[string]any
	if err := json.Unmarshal(infoOut, &manifest); err != nil {
		res.add("info: valid JSON manifest", false, err.Error())
	} else {
		missing := []string{}
		for _, k := range manifestRequired {
			if _, ok := manifest[k]; !ok {
				missing = append(missing, k)
			}
		}
		res.add("info: manifest has required fields", len(missing) == 0, strings.Join(missing, ","))
	}

	// --- list-years ---
	yearsOut, _, err := runCmd(adapterCmd, "list-years")
	if err != nil {
		res.add("list-years: exit 0", false, err.Error())
	} else {
		res.add("list-years: exit 0", true, "")
		var years []string
		if err := json.Unmarshal(yearsOut, &years); err != nil {
			res.add("list-years: JSON array of years", false, err.Error())
		} else {
			allMatch := true
			for _, y := range years {
				if !fiscalYearRe.MatchString(y) {
					allMatch = false
				}
			}
			res.add("list-years: every entry matches fiscal-year pattern", allMatch, fmt.Sprintf("%v", years))
			sortedDesc := sort.SliceIsSorted(years, func(i, j int) bool { return years[i] > years[j] })
			res.add("list-years: descending order", sortedDesc, "")
		}
	}

	// --- fetch run 1 ---
	rh1, err := fetchRun(adapterCmd, year, filepath.Join(workDir, "run1"), res, "run1")
	if err != nil {
		return res, nil
	}
	res.ResultHash = rh1

	// --- fetch run 2 (determinism) ---
	rh2, err := fetchRun(adapterCmd, year, filepath.Join(workDir, "run2"), res, "run2")
	if err != nil {
		return res, nil
	}
	res.add("resultHash deterministic across two runs", rh1 == rh2,
		fmt.Sprintf("run1=%s run2=%s", short(rh1), short(rh2)))

	return res, nil
}

// fetchRun performs one fetch, validates the document, checks raw snapshots and the
// recomputed resultHash, and returns the envelope's resultHash.
func fetchRun(adapterCmd []string, year, dir string, res *Result, label string) (string, error) {
	rawDir := filepath.Join(dir, "raw")
	outPath := filepath.Join(dir, "out.json")
	if err := os.MkdirAll(rawDir, 0o755); err != nil {
		return "", err
	}

	_, _, err := runCmd(adapterCmd, "fetch", "--year", year, "--raw-dir", rawDir, "--out", outPath)
	if err != nil {
		res.add(label+": fetch exit 0", false, err.Error())
		return "", err
	}
	res.add(label+": fetch exit 0", true, "")

	outBytes, err := os.ReadFile(outPath)
	if err != nil {
		res.add(label+": out document written", false, err.Error())
		return "", err
	}
	res.add(label+": out document written", true, "")

	// Schema validity.
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(outBytes))
	if err != nil {
		res.add(label+": out is JSON", false, err.Error())
		return "", err
	}
	if verr := contract.Validate("AdapterOutput", instance); verr != nil {
		res.add(label+": out validates against AdapterOutput", false, verr.Error())
	} else {
		res.add(label+": out validates against AdapterOutput", true, "")
	}

	// Parse for envelope + facts.
	var doc struct {
		Envelope struct {
			ResultHash   string `json:"resultHash"`
			RawSnapshots []struct {
				Sha256 string `json:"sha256"`
				Bytes  int64  `json:"bytes"`
			} `json:"rawSnapshots"`
		} `json:"envelope"`
		Facts []json.RawMessage `json:"facts"`
	}
	if err := json.Unmarshal(outBytes, &doc); err != nil {
		res.add(label+": parse envelope/facts", false, err.Error())
		return "", err
	}

	// rawHash correctness: each .bin hashes to its filename; each declared snapshot exists.
	res.add(label+": raw .bin files hash to their names", checkRawFiles(rawDir), "")
	res.add(label+": declared rawSnapshots present and correct",
		checkDeclaredSnapshots(rawDir, snapshotShas(doc.Envelope.RawSnapshots)), "")

	// resultHash recompute.
	recomputed, err := recomputeResultHash(doc.Facts)
	if err != nil {
		res.add(label+": recompute resultHash", false, err.Error())
		return doc.Envelope.ResultHash, nil
	}
	res.add(label+": envelope.resultHash matches recomputation", recomputed == doc.Envelope.ResultHash,
		fmt.Sprintf("declared=%s recomputed=%s", short(doc.Envelope.ResultHash), short(recomputed)))

	return doc.Envelope.ResultHash, nil
}

func snapshotShas(snaps []struct {
	Sha256 string `json:"sha256"`
	Bytes  int64  `json:"bytes"`
}) []string {
	out := make([]string, len(snaps))
	for i, s := range snaps {
		out[i] = s.Sha256
	}
	return out
}

// recomputeResultHash mirrors the SDKs: drop volatile fields, sort by factHash, JCS, SHA-256.
func recomputeResultHash(facts []json.RawMessage) (string, error) {
	type factMap map[string]json.RawMessage
	parsed := make([]factMap, 0, len(facts))
	for _, raw := range facts {
		var m factMap
		if err := json.Unmarshal(raw, &m); err != nil {
			return "", err
		}
		for _, k := range volatileFactFields {
			delete(m, k)
		}
		parsed = append(parsed, m)
	}
	sort.Slice(parsed, func(i, j int) bool {
		return string(parsed[i]["factHash"]) < string(parsed[j]["factHash"])
	})
	arr, err := json.Marshal(parsed)
	if err != nil {
		return "", err
	}
	canon, err := jsoncanonicalizer.Transform(arr)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(canon)
	return hex.EncodeToString(sum[:]), nil
}

func checkRawFiles(rawDir string) bool {
	entries, err := os.ReadDir(rawDir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		name := e.Name()
		if !strings.HasSuffix(name, ".bin") {
			continue
		}
		want := strings.TrimSuffix(name, ".bin")
		data, err := os.ReadFile(filepath.Join(rawDir, name))
		if err != nil {
			return false
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != want {
			return false
		}
	}
	return true
}

func checkDeclaredSnapshots(rawDir string, shas []string) bool {
	for _, sha := range shas {
		data, err := os.ReadFile(filepath.Join(rawDir, sha+".bin"))
		if err != nil {
			return false
		}
		sum := sha256.Sum256(data)
		if hex.EncodeToString(sum[:]) != sha {
			return false
		}
	}
	return true
}

func runCmd(base []string, args ...string) (stdout, stderr []byte, err error) {
	full := append(append([]string{}, base[1:]...), args...)
	cmd := exec.Command(base[0], full...)
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if e := cmd.Run(); e != nil {
		return outBuf.Bytes(), errBuf.Bytes(), fmt.Errorf("%s %v: %w (stderr: %s)", base[0], full, e, strings.TrimSpace(errBuf.String()))
	}
	return outBuf.Bytes(), errBuf.Bytes(), nil
}

func short(s string) string {
	if len(s) <= 12 {
		return s
	}
	return s[:12]
}
