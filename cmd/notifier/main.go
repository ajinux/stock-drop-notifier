package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"

	"github.com/ajinux/stock-market-notifier/internal/alert"
	"github.com/ajinux/stock-market-notifier/internal/alphavantage"
	"github.com/ajinux/stock-market-notifier/internal/config"
	"github.com/ajinux/stock-market-notifier/internal/telegram"
)

// Version information (set via ldflags during build)
var (
	version   = "dev"
	commit    = "none"
	buildDate = "unknown"
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "notifier",
	Short: "Stock market alert notifier",
	Long: `A CLI tool that monitors stock prices and sends Telegram notifications
when user-defined alert conditions are triggered.

Example usage:
  notifier check              # Evaluate all alerts and notify
  notifier check --verbose    # Show detailed evaluation results
  notifier list               # List all configured alerts
  notifier add --symbol AAPL --threshold -5 --period 7d --name "AAPL Drop"
  notifier remove "AAPL Drop"`,

	// A runtime failure is not a usage mistake — don't dump the help text after
	// one, which otherwise buries the actual error in CI logs. main already
	// prints the error, so let it own that too rather than printing it twice.
	SilenceUsage:  true,
	SilenceErrors: true,
}

var verbose bool
var checkLimit int

var checkCmd = &cobra.Command{
	Use:   "check",
	Short: "Evaluate all alerts and send notifications for triggered ones",
	Long: `Evaluates all enabled alerts against current market data.
When an alert's conditions are met, a notification is sent via Telegram.

Use --verbose to see detailed evaluation results for each alert.`,
	RunE: runCheck,
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all configured alerts",
	Long:  `Displays a table of all configured alerts with their status.`,
	RunE:  runList,
}

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new alert",
	Long: `Add a new alert configuration.

Examples:
  notifier add --symbol AAPL --threshold -5 --period 7d --name "AAPL Weekly Drop"
  notifier add --symbol IXIC --threshold -3 --period 7d --name "NASDAQ Drop"
  notifier add --symbol MSFT --threshold 10 --period 30d --name "MSFT Monthly Gain"`,
	RunE: runAdd,
}

var removeCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove an alert by name",
	Long:  `Remove an alert configuration by its name.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runRemove,
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("stock-market-notifier %s\n", version)
		fmt.Printf("  Commit: %s\n", commit)
		fmt.Printf("  Built:  %s\n", buildDate)
	},
}

var testTelegramCmd = &cobra.Command{
	Use:   "test-telegram",
	Short: "Send a mock alert notification to verify Telegram setup",
	Long: `Sends a single mock alert notification to the configured Telegram chat.
This is used to verify that your bot token, chat ID, and network connectivity are set up correctly.`,
	RunE: runTestTelegram,
}

// Flags for add command
var (
	addSymbol    string
	addThreshold float64
	addPeriod    string
	addName      string
	addEnabled   bool
	addMethod    string
)

func init() {
	// Add commands
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(removeCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(testTelegramCmd)

	// Check command flags
	checkCmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show detailed evaluation results")
	checkCmd.Flags().IntVarP(&checkLimit, "limit", "l", 0, "Limit the number of alerts to evaluate")

	// Add command flags
	addCmd.Flags().StringVarP(&addSymbol, "symbol", "s", "", "Stock symbol (e.g., AAPL, IXIC)")
	addCmd.Flags().Float64VarP(&addThreshold, "threshold", "t", 0, "Percentage threshold (negative for drops, positive for gains)")
	addCmd.Flags().StringVarP(&addPeriod, "period", "p", "", "Time period (e.g., 7d, 30d)")
	addCmd.Flags().StringVarP(&addName, "name", "n", "", "Alert name")
	addCmd.Flags().BoolVarP(&addEnabled, "enabled", "e", true, "Enable the alert")
	addCmd.Flags().StringVarP(&addMethod, "method", "m", "from_peak", "Calculation method ('from_peak' or 'period_to_period')")

	addCmd.MarkFlagRequired("symbol")
	addCmd.MarkFlagRequired("threshold")
	addCmd.MarkFlagRequired("period")
	addCmd.MarkFlagRequired("name")
}

func runCheck(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate configuration for checking
	if err := cfg.ValidateForCheck(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	// Load alerts
	alertsFile, err := config.LoadAlerts("")
	if err != nil {
		return fmt.Errorf("failed to load alerts: %w", err)
	}

	if checkLimit < 0 {
		return fmt.Errorf("limit must be a non-negative integer")
	}

	enabledAlerts := alertsFile.GetEnabledAlerts()
	if len(enabledAlerts) == 0 {
		fmt.Println("No enabled alerts configured.")
		fmt.Println("Add alerts using: notifier add --symbol AAPL --threshold -5 --period 7d --name \"My Alert\"")
		return nil
	}

	if checkLimit > 0 && checkLimit < len(enabledAlerts) {
		enabledAlerts = enabledAlerts[:checkLimit]
	}

	fmt.Printf("Evaluating %d alert(s)...\n\n", len(enabledAlerts))

	// Create Alpha Vantage client and alert engine
	avClient := alphavantage.NewClient(cfg.AlphaVantageAPIKey)
	engine := alert.NewEngine(avClient)

	// Evaluate alerts
	summary := engine.EvaluateFromConfig(enabledAlerts)

	// Display results
	if verbose {
		for _, result := range summary.Results {
			if result.IsError() {
				fmt.Printf("❌ %s: %v\n", result.Alert.Name, result.Error)
			} else {
				statusIcon := "✓ "
				if result.Triggered {
					statusIcon = "🔔"
				}

				triggerStatus := "OK"
				if result.Triggered {
					triggerStatus = "TRIGGERED"
				}

				var priceInfo string
				if result.Alert.CalculationMethod == "from_peak" || result.Alert.CalculationMethod == "" {
					extremeName := "Peak"
					if result.Alert.Threshold >= 0 {
						extremeName = "Trough"
					}
					priceInfo = fmt.Sprintf("Current: $%.2f, %s: $%.2f", result.CurrentPrice, extremeName, result.ExtremePrice)
				} else {
					priceInfo = fmt.Sprintf("Current: $%.2f, Past: $%.2f", result.CurrentPrice, result.PastPrice)
				}

				fmt.Printf("%s  %s (%s): %s (%s, Change: %.2f%%, Threshold: %.2f%%)\n",
					statusIcon, result.Alert.Name, result.Alert.Symbol, triggerStatus, priceInfo, result.PercentChange, result.Alert.Threshold)
			}
		}
		fmt.Println()
	}

	// Send notifications for triggered alerts
	triggeredResults := summary.GetTriggeredResults()
	if len(triggeredResults) > 0 {
		// Validate Telegram configuration
		if err := cfg.ValidateForNotify(); err != nil {
			fmt.Printf("Warning: Cannot send notifications - %v\n", err)
			fmt.Println("Triggered alerts:")
			for _, r := range triggeredResults {
				fmt.Printf("  - %s\n", r.Message())
			}
			return nil
		}

		// Send Telegram notifications
		tgClient := telegram.NewClient(cfg.TelegramBotToken, cfg.TelegramChatID)

		var messages []string
		for _, r := range triggeredResults {
			messages = append(messages, r.TelegramMessage())
		}

		sent, errors := tgClient.SendBatch(messages)
		fmt.Printf("Sent %d notification(s)\n", sent)

		if len(errors) > 0 {
			for _, e := range errors {
				fmt.Printf("  Error: %v\n", e)
			}
		}
	} else {
		fmt.Println("No alerts triggered.")
	}

	// Show summary
	fmt.Printf("\nSummary: %d evaluated, %d triggered, %d errors\n",
		len(summary.Results), summary.TriggeredAlerts, summary.ErrorAlerts)

	// Every alert failing means nothing was actually monitored. Exit non-zero so a
	// scheduled run goes red instead of reporting a silent success — a green check
	// on a run that checked nothing is worse than no run at all.
	if summary.ErrorAlerts > 0 && summary.ErrorAlerts == len(summary.Results) {
		return fmt.Errorf("all %d alert(s) failed to evaluate", summary.ErrorAlerts)
	}

	return nil
}

func runList(cmd *cobra.Command, args []string) error {
	alertsFile, err := config.LoadAlerts("")
	if err != nil {
		return fmt.Errorf("failed to load alerts: %w", err)
	}

	if len(alertsFile.Alerts) == 0 {
		fmt.Println("No alerts configured.")
		fmt.Println("Add alerts using: notifier add --symbol AAPL --threshold -5 --period 7d --name \"My Alert\"")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSYMBOL\tTHRESHOLD\tPERIOD\tMETHOD\tENABLED")
	fmt.Fprintln(w, "----\t------\t---------\t------\t------\t-------")

	for _, a := range alertsFile.Alerts {
		enabled := "Yes"
		if !a.Enabled {
			enabled = "No"
		}
		method := a.Condition.CalculationMethod
		if method == "" {
			method = "from_peak"
		}
		fmt.Fprintf(w, "%s\t%s\t%.1f%%\t%s\t%s\t%s\n",
			a.Name, a.Symbol, a.Condition.Threshold, a.Condition.Period, method, enabled)
	}

	w.Flush()
	return nil
}

func runAdd(cmd *cobra.Command, args []string) error {
	// Validate period format
	if _, err := config.ParsePeriod(addPeriod); err != nil {
		return fmt.Errorf("invalid period format: %s (use format like '7d' or '30d')", addPeriod)
	}

	// Validate calculation method
	if addMethod != "" && addMethod != config.CalculationMethodFromPeak && addMethod != config.CalculationMethodPeriodToPeriod {
		return fmt.Errorf("invalid calculation method: %s (must be 'from_peak' or 'period_to_period')", addMethod)
	}

	// Load existing alerts
	alertsFile, err := config.LoadAlerts("")
	if err != nil {
		return fmt.Errorf("failed to load alerts: %w", err)
	}

	// Create new alert
	newAlert := config.AlertConfig{
		Name:   addName,
		Symbol: addSymbol,
		Condition: config.AlertCondition{
			Type:              "percent_change",
			Threshold:         addThreshold,
			Period:            addPeriod,
			CalculationMethod: addMethod,
		},
		Enabled: addEnabled,
	}

	// Add to alerts
	if err := alertsFile.AddAlert(newAlert); err != nil {
		return err
	}

	// Save
	if err := config.SaveAlerts("", alertsFile); err != nil {
		return fmt.Errorf("failed to save alerts: %w", err)
	}

	fmt.Printf("Added alert: %s\n", addName)
	fmt.Printf("  Symbol: %s\n", addSymbol)
	fmt.Printf("  Threshold: %.1f%%\n", addThreshold)
	fmt.Printf("  Period: %s\n", addPeriod)
	fmt.Printf("  Method: %s\n", addMethod)
	fmt.Printf("  Enabled: %v\n", addEnabled)

	return nil
}

func runRemove(cmd *cobra.Command, args []string) error {
	name := args[0]

	// Load existing alerts
	alertsFile, err := config.LoadAlerts("")
	if err != nil {
		return fmt.Errorf("failed to load alerts: %w", err)
	}

	// Remove alert
	if err := alertsFile.RemoveAlert(name); err != nil {
		return err
	}

	// Save
	if err := config.SaveAlerts("", alertsFile); err != nil {
		return fmt.Errorf("failed to save alerts: %w", err)
	}

	fmt.Printf("Removed alert: %s\n", name)

	return nil
}

func runTestTelegram(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Validate Telegram configuration
	if err := cfg.ValidateForNotify(); err != nil {
		return fmt.Errorf("Telegram configuration invalid: %w", err)
	}

	fmt.Println("Sending mock alert to Telegram...")

	// Construct mock alert result
	mockResult := alert.AlertResult{
		Alert: alert.Alert{
			Name:              "AAPL Mock Alert (TEST)",
			Symbol:            "AAPL",
			Threshold:         -5.0,
			Period:            7,
			CalculationMethod: "from_peak",
		},
		Triggered:     true,
		CurrentPrice:  142.50,
		PastPrice:     150.00,
		ExtremePrice:  150.00,
		PercentChange: -5.0,
		EvaluatedAt:   time.Now(),
	}

	// Generate standard Telegram message
	mockMsg := mockResult.TelegramMessage()

	// Prepend clear "TEST NOTIFICATION" indicator at the very top of the Telegram message
	testHeader := "🧪 *TEST NOTIFICATION*\n" +
		"This is a test message to verify your Telegram setup.\n" +
		"-----------------------------------------\n\n"
	finalMsg := testHeader + mockMsg

	// Create Telegram client
	tgClient := telegram.NewClient(cfg.TelegramBotToken, cfg.TelegramChatID)

	// Send message
	if err := tgClient.SendMessage(finalMsg); err != nil {
		return fmt.Errorf("failed to send Telegram message: %w", err)
	}

	fmt.Println("✅ Success! Test notification sent to Telegram.")
	return nil
}
