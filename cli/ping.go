package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"time"

	"golang.org/x/xerrors"
	"tailscale.com/tailcfg"

	"cdr.dev/slog"
	"cdr.dev/slog/sloggers/sloghuman"

	"github.com/coder/pretty"

	"github.com/coder/coder/v2/cli/cliui"
	"github.com/coder/coder/v2/cli/cliutil"
	"github.com/coder/coder/v2/codersdk"
	"github.com/coder/coder/v2/codersdk/healthsdk"
	"github.com/coder/coder/v2/codersdk/workspacesdk"
	"github.com/coder/serpent"
)

func (r *RootCmd) ping() *serpent.Command {
	var (
		pingNum          int64
		pingTimeout      time.Duration
		pingWait         time.Duration
		appearanceConfig codersdk.AppearanceConfig
	)

	client := new(codersdk.Client)
	cmd := &serpent.Command{
		Annotations: workspaceCommand,
		Use:         "ping <workspace>",
		Short:       "Ping a workspace",
		Middleware: serpent.Chain(
			serpent.RequireNArgs(1),
			r.InitClient(client),
			initAppearance(client, &appearanceConfig),
		),
		Handler: func(inv *serpent.Invocation) error {
			ctx, cancel := context.WithCancel(inv.Context())
			defer cancel()

			workspaceName := inv.Args[0]
			_, workspaceAgent, err := getWorkspaceAndAgent(
				ctx, inv, client,
				false, // Do not autostart for a ping.
				workspaceName,
			)
			if err != nil {
				return err
			}

			opts := &workspacesdk.DialAgentOptions{}

			if r.verbose {
				opts.Logger = inv.Logger.AppendSinks(sloghuman.Sink(inv.Stdout)).Leveled(slog.LevelDebug)
			}

			if r.disableDirect {
				_, _ = fmt.Fprintln(inv.Stderr, "Direct connections disabled.")
				opts.BlockEndpoints = true
			}
			if !r.disableNetworkTelemetry {
				opts.EnableTelemetry = true
			}
			wsClient := workspacesdk.New(client)
			conn, err := wsClient.DialAgent(ctx, workspaceAgent.ID, opts)
			if err != nil {
				return err
			}
			defer conn.Close()

			derpMap := conn.DERPMap()
			_ = derpMap

			n := 0
			didP2p := false
			start := time.Now()
			for {
				if n > 0 {
					time.Sleep(pingWait)
				}
				n++

				ctx, cancel := context.WithTimeout(ctx, pingTimeout)
				dur, p2p, pong, err := conn.Ping(ctx)
				cancel()
				if err != nil {
					if xerrors.Is(err, context.DeadlineExceeded) {
						_, _ = fmt.Fprintf(inv.Stdout, "ping to %q timed out \n", workspaceName)
						if n == int(pingNum) {
							return nil
						}
						continue
					}
					if xerrors.Is(err, context.Canceled) {
						return nil
					}

					if err.Error() == "no matching peer" {
						continue
					}

					_, _ = fmt.Fprintf(inv.Stdout, "ping to %q failed %s\n", workspaceName, err.Error())
					if n == int(pingNum) {
						return nil
					}
					continue
				}

				dur = dur.Round(time.Millisecond)
				var via string
				if p2p {
					if !didP2p {
						_, _ = fmt.Fprintln(inv.Stdout, "p2p connection established in",
							pretty.Sprint(cliui.DefaultStyles.DateTimeStamp, time.Since(start).Round(time.Millisecond).String()),
						)
					}
					didP2p = true

					via = fmt.Sprintf("%s via %s",
						pretty.Sprint(cliui.DefaultStyles.Fuchsia, "p2p"),
						pretty.Sprint(cliui.DefaultStyles.Code, pong.Endpoint),
					)
				} else {
					derpName := "unknown"
					derpRegion, ok := derpMap.Regions[pong.DERPRegionID]
					if ok {
						derpName = derpRegion.RegionName
					}
					via = fmt.Sprintf("%s via %s",
						pretty.Sprint(cliui.DefaultStyles.Fuchsia, "proxied"),
						pretty.Sprint(cliui.DefaultStyles.Code, fmt.Sprintf("DERP(%s)", derpName)),
					)
				}

				_, _ = fmt.Fprintf(inv.Stdout, "pong from %s %s in %s\n",
					pretty.Sprint(cliui.DefaultStyles.Keyword, workspaceName),
					via,
					pretty.Sprint(cliui.DefaultStyles.DateTimeStamp, dur.String()),
				)

				if n == int(pingNum) {
					break
				}
			}
			diagCtx, diagCancel := context.WithTimeout(inv.Context(), 30*time.Second)
			defer diagCancel()
			diags := conn.GetPeerDiagnostics()
			cliui.PeerDiagnostics(inv.Stdout, diags)

			ni := conn.GetNetInfo()
			connDiags := cliui.ConnDiags{
				PingP2P:            didP2p,
				DisableDirect:      r.disableDirect,
				LocalNetInfo:       ni,
				Verbose:            r.verbose,
				TroubleshootingURL: appearanceConfig.DocsURL + "/networking/troubleshooting",
			}

			awsRanges, err := cliutil.FetchAWSIPRanges(diagCtx, cliutil.AWSIPRangesURL)
			if err != nil {
				opts.Logger.Debug(inv.Context(), "failed to retrieve AWS IP ranges", slog.Error(err))
			}

			connDiags.ClientIPIsAWS = isAWSIP(awsRanges, ni)

			connInfo, err := wsClient.AgentConnectionInfoGeneric(diagCtx)
			if err != nil || connInfo.DERPMap == nil {
				return xerrors.Errorf("Failed to retrieve connection info from server: %w\n", err)
			}
			connDiags.ConnInfo = connInfo
			ifReport, err := healthsdk.RunInterfacesReport()
			if err == nil {
				connDiags.LocalInterfaces = &ifReport
			} else {
				_, _ = fmt.Fprintf(inv.Stdout, "Failed to retrieve local interfaces report: %v\n", err)
			}

			agentNetcheck, err := conn.Netcheck(diagCtx)
			if err == nil {
				connDiags.AgentNetcheck = &agentNetcheck
				connDiags.AgentIPIsAWS = isAWSIP(awsRanges, agentNetcheck.NetInfo)
			} else {
				var sdkErr *codersdk.Error
				if errors.As(err, &sdkErr) && sdkErr.StatusCode() == http.StatusNotFound {
					_, _ = fmt.Fprint(inv.Stdout, "Could not generate full connection report as the workspace agent is outdated\n")
				} else {
					_, _ = fmt.Fprintf(inv.Stdout, "Failed to retrieve connection report from agent: %v\n", err)
				}
			}

			connDiags.Write(inv.Stdout)
			return nil
		},
	}

	cmd.Options = serpent.OptionSet{
		{
			Flag:        "wait",
			Description: "Specifies how long to wait between pings.",
			Default:     "1s",
			Value:       serpent.DurationOf(&pingWait),
		},
		{
			Flag:          "timeout",
			FlagShorthand: "t",
			Default:       "5s",
			Description:   "Specifies how long to wait for a ping to complete.",
			Value:         serpent.DurationOf(&pingTimeout),
		},
		{
			Flag:          "num",
			FlagShorthand: "n",
			Default:       "10",
			Description:   "Specifies the number of pings to perform.",
			Value:         serpent.Int64Of(&pingNum),
		},
	}
	return cmd
}

func isAWSIP(awsRanges *cliutil.AWSIPRanges, ni *tailcfg.NetInfo) bool {
	if awsRanges == nil {
		return false
	}
	if ni.GlobalV4 != "" {
		ip, err := netip.ParseAddr(ni.GlobalV4)
		if err == nil && awsRanges.CheckIP(ip) {
			return true
		}
	}
	if ni.GlobalV6 != "" {
		ip, err := netip.ParseAddr(ni.GlobalV6)
		if err == nil && awsRanges.CheckIP(ip) {
			return true
		}
	}
	return false
}
