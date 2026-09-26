// Command apidiscovery fetches the published contracts of the SAP
// Integration Suite services this provider knows ($metadata, OpenAPI),
// compares them with the snapshots in testdata/api-metadata and prints what
// changed in semantic terms. It writes snapshots and the discovery report
// only when asked to.
//
// Credentials come from the same environment variables as the provider
// (SAP_INTEGRATION_SUITE_*); run with -list to see which services are
// configured. Nothing it prints contains a host name or a secret.
//
// Examples:
//
//	go run ./cmd/apidiscovery                     # compare live contracts with the snapshots
//	go run ./cmd/apidiscovery -fail-on additive   # also fail on anything new (API gap discovery)
//	go run ./cmd/apidiscovery -update             # write changed snapshots
//	go run ./cmd/apidiscovery -from cloud-integration=.specs/cloudintegration-metadata.xml -update
//	go run ./cmd/apidiscovery -report docs/api-discovery-report.md -offline
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/apidiscovery"
	"github.com/Prideth/terraform-provider-sap-integration-suite/internal/apimeta"
)

type fromFlags map[string]string

func (f fromFlags) String() string { return fmt.Sprint(map[string]string(f)) }

func (f fromFlags) Set(v string) error {
	id, path, ok := strings.Cut(v, "=")
	if !ok {
		return fmt.Errorf("want service=file, got %q", v)
	}
	f[id] = path
	return nil
}

func main() {
	from := fromFlags{}
	var (
		services = flag.String("services", "", "comma-separated service IDs (default: all)")
		list     = flag.Bool("list", false, "list the services and whether they are configured, then exit")
		update   = flag.Bool("update", false, "write snapshots whose contract changed")
		offline  = flag.Bool("offline", false, "do not fetch anything; only use -from documents and snapshots")
		report   = flag.String("report", "", "write the discovery report to this file")
		failOn   = flag.String("fail-on", "provider-breaking", "exit non-zero on: none, provider-breaking, breaking, additive")
		rawDir   = flag.String("raw-dir", "", "also save each fetched document here (keep it outside the repository)")
	)
	flag.Var(from, "from", "use a local document for a service: service=file (repeatable)")
	flag.Parse()

	if *list {
		printServices()
		return
	}

	selected, err := selectServices(*services)
	if err != nil {
		fail(err)
	}

	exit := 0
	for _, svc := range selected {
		code := runService(svc, from[svc.ID], *offline, *update, *failOn, *rawDir)
		if code > exit {
			exit = code
		}
	}

	if *report != "" {
		if err := apidiscovery.WriteReport(*report); err != nil {
			fail(err)
		}
		fmt.Printf("\nreport written to %s\n", *report)
	}
	os.Exit(exit)
}

func selectServices(ids string) ([]apidiscovery.Service, error) {
	if ids == "" {
		return apidiscovery.Services, nil
	}
	var out []apidiscovery.Service
	for _, id := range strings.Split(ids, ",") {
		svc, ok := apidiscovery.Lookup(strings.TrimSpace(id))
		if !ok {
			return nil, fmt.Errorf("unknown service %q (see -list)", id)
		}
		out = append(out, svc)
	}
	return out, nil
}

func printServices() {
	for _, svc := range apidiscovery.Services {
		ok, missing := svc.Configured()
		status := "configured"
		if !ok {
			status = "not configured, missing " + strings.Join(missing, ", ")
		}
		fmt.Printf("%-36s %-9s %s\n    %s\n    evidence: %s\n", svc.ID, svc.Protocol, status, svc.Title, svc.Evidence)
	}
}

// runService returns 0 (fine), 1 (a change the -fail-on policy rejects) or
// 2 (the contract could not be read).
func runService(svc apidiscovery.Service, fromFile string, offline, update bool, failOn, rawDir string) int {
	fmt.Printf("\n=== %s\n", svc.ID)
	live, raw, issues, err := obtain(svc, fromFile, offline)
	if err != nil {
		fmt.Printf("SKIPPED: %v\n", err)
		return 0
	}
	if live == nil {
		fmt.Println("SKIPPED: not configured and no -from document")
		return 0
	}
	if rawDir != "" && raw != nil {
		if err := saveRaw(rawDir, svc, raw); err != nil {
			fmt.Printf("could not save the raw document: %v\n", err)
		}
	}
	for _, issue := range issues {
		fmt.Println("SERVICE DOCUMENT:", issue)
	}
	fmt.Printf("%d entity sets, %d entity types, %d complex types, %d operations; %d types reachable, %d unreachable, %d unresolved\n",
		len(live.EntitySets), len(live.EntityTypes), len(live.ComplexTypes), len(live.Operations),
		lenOrZero(live.Graph, func(g *apimeta.GraphSummary) int { return len(g.Reachable) }),
		lenOrZero(live.Graph, func(g *apimeta.GraphSummary) int { return len(g.Unreachable) }),
		lenOrZero(live.Graph, func(g *apimeta.GraphSummary) int { return len(g.Unresolved) }))

	snapshot, err := apimeta.LoadSnapshot(svc.ID)
	if err != nil {
		fmt.Println("no snapshot yet")
		if update {
			apidiscovery.CarryCapturedAt(nil, live, true)
			return write(live)
		}
		return 0
	}

	cmp := apidiscovery.Compare(snapshot, live)
	for _, p := range cmp.ContractProblems {
		fmt.Println("CONTRACT:", p)
	}
	if len(cmp.Changes) == 0 {
		fmt.Println("no changes against the snapshot")
	}
	for _, c := range cmp.Changes {
		fmt.Println(c.String())
	}
	fmt.Printf("%d changes: %d additive, %d breaking (%d touching the provider)\n",
		len(cmp.Changes), len(cmp.Additive), len(cmp.Breaking), len(cmp.ProviderBreaking))

	code := 0
	switch failOn {
	case "none":
	case "additive":
		if len(cmp.Changes) > 0 || len(cmp.ContractProblems) > 0 {
			fmt.Println("FAIL: the contract differs from the snapshot. Investigate the changes, then run with -update to accept them.")
			code = 1
		}
	case "breaking":
		if len(cmp.Breaking) > 0 || len(cmp.ContractProblems) > 0 {
			code = 1
		}
	default: // provider-breaking
		if len(cmp.ProviderBreaking) > 0 || len(cmp.ContractProblems) > 0 {
			code = 1
		}
	}

	if update && len(cmp.Changes) > 0 {
		apidiscovery.CarryCapturedAt(snapshot, live, true)
		if w := write(live); w > 0 {
			return w
		}
	}
	return code
}

func obtain(svc apidiscovery.Service, fromFile string, offline bool) (*apimeta.Service, []byte, []string, error) {
	if fromFile != "" {
		data, err := os.ReadFile(fromFile) //nolint:gosec // G304: file named on the command line
		if err != nil {
			return nil, nil, nil, err
		}
		var s *apimeta.Service
		if svc.Protocol == apimeta.ProtocolOpenAPI {
			s, err = apimeta.ParseOpenAPI(svc.ID, data)
		} else {
			s, err = apimeta.ParseEDMX(svc.ID, data)
		}
		if err != nil {
			return nil, nil, nil, err
		}
		s.Source = "local document"
		return s, nil, nil, nil
	}
	if ok, _ := svc.Configured(); offline || !ok {
		return nil, nil, nil, nil
	}
	res, err := apidiscovery.Fetch(context.Background(), svc)
	if err != nil {
		return nil, nil, nil, err
	}
	return res.Service, res.Raw, res.ServiceDocumentIssues, nil
}

func write(s *apimeta.Service) int {
	if err := apimeta.WriteSnapshot(s); err != nil {
		fmt.Printf("could not write the snapshot: %v\n", err)
		return 2
	}
	fmt.Printf("snapshot written: testdata/api-metadata/%s.json\n", s.ID)
	return 0
}

func saveRaw(dir string, svc apidiscovery.Service, raw []byte) error {
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	ext := ".xml"
	if svc.Protocol == apimeta.ProtocolOpenAPI {
		ext = ".json"
	}
	return os.WriteFile(filepath.Join(dir, svc.ID+"-metadata"+ext), raw, 0o600)
}

func lenOrZero(g *apimeta.GraphSummary, f func(*apimeta.GraphSummary) int) int {
	if g == nil {
		return 0
	}
	return f(g)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(2)
}
