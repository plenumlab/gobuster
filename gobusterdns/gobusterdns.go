package gobusterdns

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"net"
	"strings"
	"text/tabwriter"

	"github.com/OJ/gobuster/libgobuster"
	"github.com/google/uuid"
)

// GobusterDNS is the main type to implement the interface
type GobusterDNS struct {
	globalopts  *libgobuster.Options
	options     *OptionsDNS
	isWildcard  bool
	wildcardIps libgobuster.StringSet
}

// NewGobusterDNS creates a new initialized GobusterDNS
func NewGobusterDNS(globalopts *libgobuster.Options, opts *OptionsDNS) (*GobusterDNS, error) {
	if globalopts == nil {
		return nil, fmt.Errorf("please provide valid global options")
	}

	if opts == nil {
		return nil, fmt.Errorf("please provide valid plugin options")
	}

	g := GobusterDNS{
		options:     opts,
		globalopts:  globalopts,
		wildcardIps: libgobuster.NewStringSet(),
	}
	return &g, nil
}

// PreRun is the pre run implementation of gobusterdns
func (d *GobusterDNS) PreRun() error {
	// Resolve a subdomain sthat probably shouldn't exist
	guid := uuid.New()
	wildcardIps, err := dnsLookup(fmt.Sprintf("%s.%s", guid, d.options.Domain))
	if err == nil {
		d.isWildcard = true
		d.wildcardIps.AddRange(wildcardIps)
		log.Printf("[-] Wildcard DNS found. IP address(es): %s", d.wildcardIps.Stringify())
		if !d.options.WildcardForced {
			return fmt.Errorf("To force processing of Wildcard DNS, specify the '--wildcard' switch.")
		}
	}

	if !d.globalopts.Quiet {
		// Provide a warning if the base domain doesn't resolve (in case of typo)
		_, err = dnsLookup(d.options.Domain)
		if err != nil {
			// Not an error, just a warning. Eg. `yp.to` doesn't resolve, but `cr.py.to` does!
			log.Printf("[-] Unable to validate base domain: %s", d.options.Domain)
		}
	}

	return nil
}

// Run is the process implementation of gobusterdns
func (d *GobusterDNS) Run(word string) ([]libgobuster.Result, error) {
	subdomain := fmt.Sprintf("%s.%s", word, d.options.Domain)
	ips, err := dnsLookup(subdomain)
	var ret []libgobuster.Result
	if err == nil {
		if !d.isWildcard || !d.wildcardIps.ContainsAny(ips) {
			result := libgobuster.Result{
				Entity: subdomain,
			}
			if d.options.ShowIPs {
				result.Extra = strings.Join(ips, ", ")
			} else if d.options.ShowCNAME {
				cname, err := dnsLookupCname(subdomain)
				if err == nil {
					result.Extra = cname
				}
			}
			ret = append(ret, result)
		}
	} else if d.globalopts.Verbose {
		ret = append(ret, libgobuster.Result{
			Entity: subdomain,
			Status: 404,
		})
	}
	return ret, nil
}

// ResultToString is the to string implementation of gobusterdns
func (d *GobusterDNS) ResultToString(r *libgobuster.Result) (*string, error) {
	buf := &bytes.Buffer{}

	if r.Status == 404 {
		if _, err := fmt.Fprintf(buf, "Missing: %s\n", r.Entity); err != nil {
			return nil, err
		}
	} else if d.options.ShowIPs {
		if _, err := fmt.Fprintf(buf, "Found: %s [%s]\n", r.Entity, r.Extra); err != nil {
			return nil, err
		}
	} else if d.options.ShowCNAME {
		if _, err := fmt.Fprintf(buf, "Found: %s [%s]\n", r.Entity, r.Extra); err != nil {
			return nil, err
		}
	} else {
		if _, err := fmt.Fprintf(buf, "Found: %s\n", r.Entity); err != nil {
			return nil, err
		}
	}

	s := buf.String()
	return &s, nil
}

// GetConfigString returns the string representation of the current config
func (d *GobusterDNS) GetConfigString() (string, error) {
	var buffer bytes.Buffer
	bw := bufio.NewWriter(&buffer)
	tw := tabwriter.NewWriter(bw, 0, 5, 3, ' ', 0)
	o := d.options
	if _, err := fmt.Fprintf(tw, "[+] Domain:\t%s\n", o.Domain); err != nil {
		return "", err
	}
	if _, err := fmt.Fprintf(tw, "[+] Threads:\t%d\n", d.globalopts.Threads); err != nil {
		return "", err
	}

	if o.ShowCNAME {
		if _, err := fmt.Fprintf(tw, "[+] Show CNAME:\ttrue\n"); err != nil {
			return "", err
		}
	}

	if o.ShowIPs {
		if _, err := fmt.Fprintf(tw, "[+] Show IPs:\ttrue\n"); err != nil {
			return "", err
		}
	}

	if o.WildcardForced {
		if _, err := fmt.Fprintf(tw, "[+] Wildcard forced:\ttrue\n"); err != nil {
			return "", err
		}
	}

	wordlist := "stdin (pipe)"
	if d.globalopts.Wordlist != "-" {
		wordlist = d.globalopts.Wordlist
	}
	if _, err := fmt.Fprintf(tw, "[+] Wordlist:\t%s\n", wordlist); err != nil {
		return "", err
	}

	if d.globalopts.Verbose {
		if _, err := fmt.Fprintf(tw, "[+] Verbose:\ttrue\n"); err != nil {
			return "", err
		}
	}

	if err := tw.Flush(); err != nil {
		return "", fmt.Errorf("error on tostring: %v", err)
	}

	if err := bw.Flush(); err != nil {
		return "", fmt.Errorf("error on tostring: %v", err)
	}

	return strings.TrimSpace(buffer.String()), nil
}

func dnsLookup(domain string) ([]string, error) {
	return net.LookupHost(domain)
}

func dnsLookupCname(domain string) (string, error) {
	return net.LookupCNAME(domain)
}
