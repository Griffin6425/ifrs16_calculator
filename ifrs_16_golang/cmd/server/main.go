package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"ifrs16_calculator/internal/calculation"
	"ifrs16_calculator/internal/platform/export"
	"ifrs16_calculator/internal/platform/parsing"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxUploadSize = 10 * 1024 * 1024 // 10 MB

// CalculationResult holds the calculated outputs for a single lease.
type CalculationResult struct {
	LeaseID           string                          `json:"leaseId"`
	InitialLiability  float64                         `json:"initialLiability"`
	InitialRoUAsset   float64                         `json:"initialRoUAsset"`
	DiscountRate      float64                         `json:"discountRate"`     // Added discount rate
	PaymentAmount     float64                         `json:"paymentAmount"`    // Added payment amount
	PaymentFrequency  string                          `json:"paymentFrequency"` // Added payment frequency
	StartDate         string                          `json:"startDate"`        // Added start date
	EndDate           string                          `json:"endDate"`          // Added end date
	LiabilitySchedule []calculation.AmortizationEntry `json:"liabilitySchedule"`
	RoUAssetSchedule  []calculation.AmortizationEntry `json:"rouAssetSchedule"`
	// 账期摘要信息
	AccountingPeriodStart  string  `json:"accountingPeriodStart,omitempty"`  // 账期开始日期
	AccountingPeriodEnd    string  `json:"accountingPeriodEnd,omitempty"`    // 账期结束日期
	PeriodLiabilityStart   float64 `json:"periodLiabilityStart,omitempty"`   // 账期期初负债
	PeriodLiabilityEnd     float64 `json:"periodLiabilityEnd,omitempty"`     // 账期期末负债
	PeriodRoUAssetStart    float64 `json:"periodRoUAssetStart,omitempty"`    // 账期期初使用权资产
	PeriodRoUAssetEnd      float64 `json:"periodRoUAssetEnd,omitempty"`      // 账期期末使用权资产
	PeriodInterestExpense  float64 `json:"periodInterestExpense,omitempty"`  // 账期内利息费用总额
	PeriodDepreciation     float64 `json:"periodDepreciation,omitempty"`     // 账期内折旧费用总额
	PeriodPayments         float64 `json:"periodPayments,omitempty"`         // 账期内付款总额
	PeriodPrincipalPayment float64 `json:"periodPrincipalPayment,omitempty"` // 添加本金偿还金额到结果中
	Error                  string  `json:"error,omitempty"`                  // To report errors for specific leases
}

// PageData holds the data for rendering templates
type PageData struct {
	Active string
	Data   interface{}
}

func main() {
	// Print working directory for debugging
	cwd, _ := filepath.Abs(".")
	log.Printf("Current working directory: %s", cwd)

	// Check if template directory exists
	templateDir := filepath.Join("..", "..", "web", "templates")
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		log.Printf("Template directory not found at: %s", templateDir)
	} else {
		log.Printf("Template directory found at: %s", templateDir)
		// List files in template directory
		files, err := os.ReadDir(templateDir)
		if err != nil {
			log.Printf("Error reading template directory: %v", err)
		} else {
			log.Printf("Files in template directory:")
			for _, file := range files {
				log.Printf("- %s", file.Name())
			}
		}
	}

	// Create a custom template map for each page
	layoutPath := filepath.Join("..", "..", "web", "templates", "layout.html")
	homePath := filepath.Join("..", "..", "web", "templates", "home.html")
	calcPath := filepath.Join("..", "..", "web", "templates", "calculate.html")
	docsPath := filepath.Join("..", "..", "web", "templates", "documentation.html")
	troublePath := filepath.Join("..", "..", "web", "templates", "troubleshoot.html")

	// Setup template for each page
	homeTemplate := template.Must(template.ParseFiles(layoutPath, homePath))
	calcTemplate := template.Must(template.ParseFiles(layoutPath, calcPath))
	docsTemplate := template.Must(template.ParseFiles(layoutPath, docsPath))
	troubleTemplate := template.Must(template.ParseFiles(layoutPath, troublePath))

	log.Printf("Templates loaded successfully")

	// Setup static file server with absolute path
	staticDir := filepath.Join("..", "..", "web", "static")
	absStaticPath, err := filepath.Abs(staticDir)
	if err != nil {
		log.Printf("Error resolving static path: %v", err)
		// Fallback to relative path
		absStaticPath = "web/static"
	}
	log.Printf("Serving static files from: %s", absStaticPath)
	fs := http.FileServer(http.Dir(absStaticPath))

	// Try to find an available port
	startPort := 8080
	maxAttempts := 10

	// Create a custom handler for all routes
	mux := http.NewServeMux()

	// Setup static file server on our custom mux
	mux.Handle("/static/", http.StripPrefix("/static/", fs))

	// Add a route specifically for debugging templates
	mux.HandleFunc("/debug-templates", func(w http.ResponseWriter, r *http.Request) {
		templates := homeTemplate.Templates()
		w.Header().Set("Content-Type", "text/plain")
		fmt.Fprintf(w, "Loaded %d templates:\n", len(templates))
		for _, t := range templates {
			fmt.Fprintf(w, "- %s\n", t.Name())
		}
	})

	// Register routes on our custom mux
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		data := PageData{
			Active: "home",
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := homeTemplate.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Printf("Error rendering template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/calculate", func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Calculate endpoint called with method: %s", r.Method)

		if r.Method == http.MethodGet {
			data := PageData{
				Active: "calculate",
			}

			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := calcTemplate.ExecuteTemplate(w, "layout.html", data); err != nil {
				log.Printf("Error rendering template: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
			return
		}

		handleCalculate(w, r)
	})

	mux.HandleFunc("/documentation", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Active: "docs",
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := docsTemplate.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Printf("Error rendering template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/troubleshoot", func(w http.ResponseWriter, r *http.Request) {
		data := PageData{
			Active: "troubleshoot",
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := troubleTemplate.ExecuteTemplate(w, "layout.html", data); err != nil {
			log.Printf("Error rendering template: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
	})

	mux.HandleFunc("/export", handleExport)

	// Try ports until one works
	for attempt := 0; attempt < maxAttempts; attempt++ {
		port := startPort + attempt
		portStr := fmt.Sprintf("%d", port)

		fmt.Printf("Attempting to start server on http://localhost:%s\n", portStr)

		// Start with a non-blocking check first
		listener, listenErr := net.Listen("tcp", ":"+portStr)
		if listenErr != nil {
			fmt.Printf("Port %s is not available, trying next port\n", portStr)
			continue
		}

		listener.Close()

		// Now try to start the server with our custom mux
		server := &http.Server{
			Addr:    ":" + portStr,
			Handler: mux,
		}
		fmt.Printf("Server starting on http://localhost:%s\n", portStr)

		log.Fatal(server.ListenAndServe())
		return // This will never be reached due to log.Fatal above
	}

	log.Fatalf("Could not start server after %d attempts", maxAttempts)
}

func handleCalculate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	log.Printf("Processing calculation request from %s", r.RemoteAddr)

	// Set max upload size
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	err := r.ParseMultipartForm(maxUploadSize)
	if err != nil {
		log.Printf("Error parsing multipart form: %v", err)
		sendJSONError(w, fmt.Sprintf("File too large or form parsing error: %v", err), http.StatusBadRequest)
		return
	}

	// Check if the form was properly parsed
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		log.Printf("No multipart form data found")
		sendJSONError(w, "No file upload data found in request", http.StatusBadRequest)
		return
	}

	// Get file from form
	fileHeaders := r.MultipartForm.File["leaseFile"]
	if len(fileHeaders) == 0 {
		log.Printf("No 'leaseFile' field found in form")
		sendJSONError(w, "No file was uploaded. Please select a file to upload.", http.StatusBadRequest)
		return
	}

	handler := fileHeaders[0]
	file, err := handler.Open()
	if err != nil {
		log.Printf("Error opening uploaded file: %v", err)
		sendJSONError(w, fmt.Sprintf("Error retrieving the file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Determine file type
	fileExt := strings.ToLower(filepath.Ext(handler.Filename))
	var fileType string
	if fileExt == ".csv" {
		fileType = "csv"
	} else if fileExt == ".xlsx" {
		fileType = "xlsx"
	} else {
		log.Printf("Invalid file type: %s", fileExt)
		sendJSONError(w, "Invalid file type. Only .csv and .xlsx are supported.", http.StatusBadRequest)
		return
	}

	log.Printf("Received file: %s, Type: %s, Size: %d bytes", handler.Filename, fileType, handler.Size)

	// Check if we should skip header
	skipHeader := r.FormValue("skipHeader") == "on"
	log.Printf("Skip header option: %v", skipHeader)

	// 获取账期日期(如果提供)
	accountingPeriodStart := r.FormValue("accountingPeriodStart")
	accountingPeriodEnd := r.FormValue("accountingPeriodEnd")
	hasAccountingPeriod := accountingPeriodStart != "" && accountingPeriodEnd != ""

	if hasAccountingPeriod {
		log.Printf("账期设置: %s 至 %s", accountingPeriodStart, accountingPeriodEnd)
	}

	// Parse the file
	parseConfig := parsing.ParseConfig{SkipHeader: skipHeader}
	parsedLeases, err := parsing.ParseLeasesFromFile(file, fileType, parseConfig)
	if err != nil {
		log.Printf("Error parsing file: %v", err)
		sendJSONError(w, fmt.Sprintf("Error parsing file: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("Successfully parsed %d leases.", len(parsedLeases))

	// Process each lease
	results := make([]CalculationResult, 0, len(parsedLeases))
	for _, l := range parsedLeases {
		result := CalculationResult{
			LeaseID:          l.ID,
			DiscountRate:     l.DiscountRate,                   // Store the discount rate from the lease
			PaymentAmount:    l.PaymentAmount,                  // Store the payment amount directly
			PaymentFrequency: string(l.PaymentFrequency),       // Store the payment frequency directly
			StartDate:        l.StartDate.Format("2006-01-02"), // Store the start date directly
			EndDate:          l.EndDate.Format("2006-01-02"),   // Store the end date directly
		}

		liability, err := calculation.CalculateLeaseLiability(l)
		if err != nil {
			log.Printf("Error calculating liability for lease %s: %v", l.ID, err)
			result.Error = fmt.Sprintf("Liability calculation error: %v", err)
			results = append(results, result)
			continue // Skip to next lease if initial calc fails
		}
		result.InitialLiability = liability

		rouAsset, err := calculation.CalculateInitialRoUAsset(liability, l) // Assuming RoU = Liability for now
		if err != nil {
			log.Printf("Error calculating RoU asset for lease %s: %v", l.ID, err)
			result.Error = fmt.Sprintf("RoU asset calculation error: %v", err)
			results = append(results, result)
			continue
		}
		result.InitialRoUAsset = rouAsset

		liabSchedule, err := calculation.GenerateLiabilitySchedule(l, liability)
		if err != nil {
			log.Printf("Error generating liability schedule for lease %s: %v", l.ID, err)
			result.Error = fmt.Sprintf("Liability schedule generation error: %v", err)
			// Still include initial values even if schedule fails?
			results = append(results, result)
			continue
		}
		result.LiabilitySchedule = liabSchedule

		rouSchedule, err := calculation.GenerateRoUAssetSchedule(l, rouAsset)
		if err != nil {
			log.Printf("Error generating RoU asset schedule for lease %s: %v", l.ID, err)
			result.Error = fmt.Sprintf("RoU asset schedule generation error: %v", err)
			// Still include initial values & liab schedule?
			results = append(results, result)
			continue
		}
		result.RoUAssetSchedule = rouSchedule

		// Update the start/end dates from the schedule ONLY if they weren't set properly from the lease
		if result.StartDate == "0001-01-01" && len(liabSchedule) > 0 {
			firstPeriodDate := liabSchedule[0].Date
			result.StartDate = firstPeriodDate.Format("2006-01-02")
		}

		if result.EndDate == "0001-01-01" && len(liabSchedule) > 0 {
			lastIdx := len(liabSchedule) - 1
			endDate := liabSchedule[lastIdx].Date
			result.EndDate = endDate.Format("2006-01-02")
		}

		// 如果提供了账期范围,计算账期摘要
		if hasAccountingPeriod {
			if err := calculateAccountingPeriodSummary(&result, accountingPeriodStart, accountingPeriodEnd); err != nil {
				log.Printf("计算账期摘要时出错: %v", err)
				// 不需要中断,将错误添加到结果中即可
				if result.Error == "" {
					result.Error = fmt.Sprintf("账期摘要计算错误: %v", err)
				} else {
					result.Error += fmt.Sprintf("; 账期摘要计算错误: %v", err)
				}
			}
		}

		results = append(results, result)
	}

	log.Printf("Processed %d leases, returning results.", len(results))

	// Check if request is AJAX (JSON) or form post
	if strings.Contains(r.Header.Get("Accept"), "application/json") ||
		strings.Contains(r.Header.Get("Content-Type"), "application/json") ||
		r.Header.Get("X-Requested-With") == "XMLHttpRequest" {
		// Send results as JSON for AJAX requests
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(results); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
			if w.Header().Get("Content-Type") == "" {
				sendJSONError(w, "Failed to encode results to JSON", http.StatusInternalServerError)
			}
		}
	} else {
		// For regular form posts, redirect to a results page
		// For now, just serve JSON as we don't have a dedicated results page
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(results); err != nil {
			log.Printf("Error encoding JSON response: %v", err)
			http.Error(w, "Failed to encode results to JSON", http.StatusInternalServerError)
		}
	}
}

// handleExport handles the export of calculation results to Excel
func handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		sendJSONError(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse the request body
	var requestData []CalculationResult
	if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
		sendJSONError(w, fmt.Sprintf("Error parsing request body: %v", err), http.StatusBadRequest)
		return
	}

	if len(requestData) == 0 {
		sendJSONError(w, "No calculation results to export", http.StatusBadRequest)
		return
	}

	// Convert calculation results to export format
	exportResults := make([]export.LeaseResultExport, 0, len(requestData))
	for _, result := range requestData {
		// Skip items with errors
		if result.Error != "" {
			continue
		}

		// Create LeaseResultExport from CalculationResult
		startDate, _ := time.Parse("2006-01-02", result.StartDate)
		endDate, _ := time.Parse("2006-01-02", result.EndDate)

		// Calculate lease term in years
		days := endDate.Sub(startDate).Hours() / 24
		leaseTerm := days / 365.25 // Using 365.25 to account for leap years

		exportResult := export.LeaseResultExport{
			LeaseID:           result.LeaseID,
			StartDate:         startDate,
			EndDate:           endDate,
			PaymentAmount:     result.PaymentAmount,    // Direct from result
			PaymentFrequency:  result.PaymentFrequency, // Direct from result
			DiscountRate:      result.DiscountRate,     // Direct from result
			InitialLiability:  result.InitialLiability,
			InitialRoUAsset:   result.InitialRoUAsset,
			LiabilitySchedule: result.LiabilitySchedule,
			RoUAssetSchedule:  result.RoUAssetSchedule,
			LeaseTerm:         leaseTerm, // Add lease term in years
			// 添加账期摘要信息
			AccountingPeriodStart:  result.AccountingPeriodStart,
			AccountingPeriodEnd:    result.AccountingPeriodEnd,
			PeriodLiabilityStart:   result.PeriodLiabilityStart,
			PeriodLiabilityEnd:     result.PeriodLiabilityEnd,
			PeriodRoUAssetStart:    result.PeriodRoUAssetStart,
			PeriodRoUAssetEnd:      result.PeriodRoUAssetEnd,
			PeriodInterestExpense:  result.PeriodInterestExpense,
			PeriodDepreciation:     result.PeriodDepreciation,
			PeriodPayments:         result.PeriodPayments,
			PeriodPrincipalPayment: result.PeriodPrincipalPayment,
		}

		exportResults = append(exportResults, exportResult)
	}

	// Generate Excel file
	excelBytes, err := export.ExportToExcel(exportResults)
	if err != nil {
		sendJSONError(w, fmt.Sprintf("Error generating Excel file: %v", err), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", "attachment; filename=ifrs16_calculation_results.xlsx")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(excelBytes)))

	// Write Excel data to response
	w.Write(excelBytes)
}

// sendJSONError is a helper to return errors as JSON responses.
func sendJSONError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	errorResponse := map[string]string{"error": message}
	if err := json.NewEncoder(w).Encode(errorResponse); err != nil {
		// Log error internally if we can't even send the JSON error message
		log.Printf("Error encoding JSON error response: %v", err)
	}
}

// calculateAccountingPeriodSummary 计算指定账期的摘要数据
func calculateAccountingPeriodSummary(result *CalculationResult, periodStart, periodEnd string) error {
	// 解析日期
	start, err := time.Parse("2006-01-02", periodStart)
	if err != nil {
		return fmt.Errorf("无效的账期开始日期: %v", err)
	}

	end, err := time.Parse("2006-01-02", periodEnd)
	if err != nil {
		return fmt.Errorf("无效的账期结束日期: %v", err)
	}

	// 验证日期范围
	if end.Before(start) {
		return fmt.Errorf("账期结束日期不能早于开始日期")
	}

	// 设置账期信息
	result.AccountingPeriodStart = periodStart
	result.AccountingPeriodEnd = periodEnd

	// 处理租赁负债表
	if len(result.LiabilitySchedule) > 0 {
		var startBalance, endBalance, totalPayments, totalInterest, totalPrincipal float64
		var startFound, endFound bool

		// 寻找账期起始的账面值
		for i, entry := range result.LiabilitySchedule {
			entryDate := entry.Date

			// 寻找账期开始日期的账面值(使用最接近的前一个日期)
			if !startFound && (entryDate.Equal(start) || entryDate.After(start)) {
				if i > 0 && entryDate.After(start) {
					// 使用前一个条目的收盘余额作为起始余额
					startBalance = result.LiabilitySchedule[i-1].ClosingBalance
				} else {
					startBalance = entry.OpeningBalance
				}
				startFound = true
			}

			// 累计账期内的数据
			if (entryDate.Equal(start) || entryDate.After(start)) &&
				(entryDate.Equal(end) || entryDate.Before(end)) {

				// 累计利息费用（每天都要累计）
				totalInterest += entry.InterestExpense

				// 只在付款日累计付款和本金
				if entry.Payment > 0 {
					totalPayments += entry.Payment
					totalPrincipal += entry.PrincipalRepayment
				}
			}

			// 寻找账期结束日期的账面值(使用最接近的前一个日期的收盘余额)
			if entryDate.Equal(end) || (entryDate.After(end) && !endFound) {
				if entryDate.After(end) && i > 0 {
					// 使用前一个条目的收盘余额
					endBalance = result.LiabilitySchedule[i-1].ClosingBalance
				} else {
					// 如果正好等于结束日期，使用当天的收盘余额
					endBalance = entry.ClosingBalance
				}
				endFound = true
			}
		}

		// 如果未找到结束值,使用最后一个条目的收盘余额
		if !endFound && len(result.LiabilitySchedule) > 0 {
			lastEntry := result.LiabilitySchedule[len(result.LiabilitySchedule)-1]
			endBalance = lastEntry.ClosingBalance
		}

		// 验证本金与余额变动的一致性
		balanceChange := startBalance - endBalance
		if math.Abs(balanceChange-totalPrincipal) > 0.01 {
			// 如果有差异，使用余额变动作为本金
			totalPrincipal = balanceChange
		}

		// 验证本金+利息=付款总额
		if math.Abs((totalPrincipal+totalInterest)-totalPayments) > 0.01 {
			// 如果有差异，调整利息使总额相等
			totalInterest = totalPayments - totalPrincipal
		}

		// 使用math.Round来四舍五入数值到2位小数
		roundTo2Decimals := func(val float64) float64 {
			return math.Round(val*100) / 100
		}

		result.PeriodLiabilityStart = roundTo2Decimals(startBalance)
		result.PeriodLiabilityEnd = roundTo2Decimals(endBalance)
		result.PeriodInterestExpense = roundTo2Decimals(totalInterest)
		result.PeriodPayments = roundTo2Decimals(totalPayments)
		result.PeriodPrincipalPayment = roundTo2Decimals(totalPrincipal)
	}

	// 处理使用权资产表
	if len(result.RoUAssetSchedule) > 0 {
		var startBalance, endBalance, totalDepreciation float64
		var startFound, endFound bool

		// 寻找账期起始的账面值
		for i, entry := range result.RoUAssetSchedule {
			entryDate := entry.Date

			// 寻找账期开始日期的账面值
			if !startFound && (entryDate.Equal(start) || entryDate.After(start)) {
				if i > 0 && entryDate.After(start) {
					// 使用前一个条目的收盘余额作为起始余额
					startBalance = result.RoUAssetSchedule[i-1].ClosingBalance
				} else {
					startBalance = entry.OpeningBalance
				}
				startFound = true
			}

			// 累计账期内的折旧
			if (entryDate.Equal(start) || entryDate.After(start)) &&
				(entryDate.Equal(end) || entryDate.Before(end)) {
				totalDepreciation += entry.Depreciation
			}

			// 寻找账期结束日期的账面值
			if entryDate.Equal(end) || (entryDate.After(end) && !endFound) {
				if entryDate.After(end) && i > 0 {
					// 使用前一个条目的收盘余额
					endBalance = result.RoUAssetSchedule[i-1].ClosingBalance
				} else {
					// 如果正好等于结束日期，使用当天的收盘余额
					endBalance = entry.ClosingBalance
				}
				endFound = true
			}
		}

		// 如果未找到结束值,使用最后一个条目的收盘余额
		if !endFound && len(result.RoUAssetSchedule) > 0 {
			lastEntry := result.RoUAssetSchedule[len(result.RoUAssetSchedule)-1]
			endBalance = lastEntry.ClosingBalance
		}

		// 验证折旧总额与余额变动的一致性
		calculatedDepreciation := startBalance - endBalance
		if math.Abs(calculatedDepreciation-totalDepreciation) > 0.01 {
			totalDepreciation = calculatedDepreciation
		}

		// 使用math.Round来四舍五入数值到2位小数
		roundTo2Decimals := func(val float64) float64 {
			return math.Round(val*100) / 100
		}

		result.PeriodRoUAssetStart = roundTo2Decimals(startBalance)
		result.PeriodRoUAssetEnd = roundTo2Decimals(endBalance)
		result.PeriodDepreciation = roundTo2Decimals(totalDepreciation)
	}

	return nil
}
