package export

import (
	"fmt"
	"ifrs16_calculator/internal/calculation"
	"log"
	"time"

	"github.com/xuri/excelize/v2"
)

// LeaseResultExport contains all calculation results for a single lease
type LeaseResultExport struct {
	LeaseID           string
	StartDate         time.Time
	EndDate           time.Time
	PaymentAmount     float64
	PaymentFrequency  string
	DiscountRate      float64
	InitialLiability  float64
	InitialRoUAsset   float64
	LiabilitySchedule []calculation.AmortizationEntry
	RoUAssetSchedule  []calculation.AmortizationEntry
	// 账期摘要信息
	AccountingPeriodStart  string  // 账期开始日期
	AccountingPeriodEnd    string  // 账期结束日期
	PeriodLiabilityStart   float64 // 账期期初负债
	PeriodLiabilityEnd     float64 // 账期期末负债
	PeriodRoUAssetStart    float64 // 账期期初使用权资产
	PeriodRoUAssetEnd      float64 // 账期期末使用权资产
	PeriodInterestExpense  float64 // 账期内利息费用总额
	PeriodDepreciation     float64 // 账期内折旧费用总额
	PeriodPayments         float64 // 账期内付款总额
	PeriodPrincipalPayment float64 // 账期内本金偿还总额
	LeaseTerm              float64 // 租赁期(年)
}

// ExportToExcel creates an Excel file with the calculation results
// Returns the Excel file as a byte array
func ExportToExcel(results []LeaseResultExport) ([]byte, error) {
	if len(results) == 0 {
		return nil, fmt.Errorf("no results to export")
	}

	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			fmt.Println("Error when closing file:", err)
		}
	}()

	// Create a summary sheet
	summarySheet := "Summary"
	f.SetSheetName("Sheet1", summarySheet) // Rename default sheet

	// Set headers for summary
	headers := []string{"Lease ID", "Start Date", "End Date", "Payment", "Frequency",
		"Discount Rate", "Initial Liability", "Initial RoU Asset"}

	for i, header := range headers {
		cell := fmt.Sprintf("%c%d", 'A'+i, 1)
		f.SetCellValue(summarySheet, cell, header)
	}

	// Apply basic formatting to headers
	headerStyle, err := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#E0EBF5"}, Pattern: 1},
	})
	if err != nil {
		log.Printf("Warning: Failed to create header style: %v", err)
	}
	f.SetCellStyle(summarySheet, "A1", string(rune('A'+len(headers)-1))+"1", headerStyle)

	// Add data to summary sheet
	for i, result := range results {
		row := i + 2 // Row 1 is for headers
		f.SetCellValue(summarySheet, fmt.Sprintf("A%d", row), result.LeaseID)
		f.SetCellValue(summarySheet, fmt.Sprintf("B%d", row), result.StartDate.Format("2006-01-02"))
		f.SetCellValue(summarySheet, fmt.Sprintf("C%d", row), result.EndDate.Format("2006-01-02"))
		f.SetCellValue(summarySheet, fmt.Sprintf("D%d", row), result.PaymentAmount)
		f.SetCellValue(summarySheet, fmt.Sprintf("E%d", row), result.PaymentFrequency)
		f.SetCellValue(summarySheet, fmt.Sprintf("F%d", row), result.DiscountRate)
		f.SetCellValue(summarySheet, fmt.Sprintf("G%d", row), result.InitialLiability)
		f.SetCellValue(summarySheet, fmt.Sprintf("H%d", row), result.InitialRoUAsset)
	}

	// Format numeric cells
	numStyle, err := f.NewStyle(&excelize.Style{
		NumFmt: 4, // Financial format with 2 decimal places
	})
	if err != nil {
		log.Printf("Warning: Failed to create number style: %v", err)
	}
	f.SetCellStyle(summarySheet, "D2", fmt.Sprintf("D%d", len(results)+1), numStyle)
	f.SetCellStyle(summarySheet, "F2", fmt.Sprintf("H%d", len(results)+1), numStyle)

	// Auto-fit columns (approximate method since excelize doesn't have direct auto-fit)
	for i := range headers {
		col := string(rune('A' + i))
		f.SetColWidth(summarySheet, col, col, 15)
	}

	// Create detail sheets for each lease
	for i, result := range results {
		// Create sheet for this lease
		sheetName := fmt.Sprintf("Lease_%s", result.LeaseID)
		// Sheet names are limited to 31 chars and can't contain certain characters
		if len(sheetName) > 31 {
			sheetName = fmt.Sprintf("Lease_%d", i+1)
		}

		// Add new sheet
		_, err := f.NewSheet(sheetName)
		if err != nil {
			continue // Skip this sheet if there's an error
		}

		// 检查是否有账期摘要信息,如果有则优先添加到最上方
		hasAccountingPeriod := result.AccountingPeriodStart != "" && result.AccountingPeriodEnd != ""
		baseRow := 1 // 基础行号,如果有账期摘要则后续内容向下移动

		if hasAccountingPeriod {
			// 添加账期摘要部分到最上方
			f.SetCellValue(sheetName, "A1", "账期摘要报表")

			// 添加账期范围摘要信息
			f.SetCellValue(sheetName, "A2", "开始日期")
			f.SetCellValue(sheetName, "B2", result.StartDate.Format("2006-01-02"))
			f.SetCellValue(sheetName, "A3", "结束日期")
			f.SetCellValue(sheetName, "B3", result.EndDate.Format("2006-01-02"))
			f.SetCellValue(sheetName, "A4", "租赁期")
			f.SetCellValue(sheetName, "B4", fmt.Sprintf("%.2f年", result.LeaseTerm))
			f.SetCellValue(sheetName, "A5", "贴现率")
			f.SetCellValue(sheetName, "B5", fmt.Sprintf("%.2f%%", result.DiscountRate*100))

			// 空行
			f.SetCellValue(sheetName, "A7", "租赁资产负债表现摘要")

			// 计算租赁负债变动的组成部分
			principalPayment := result.PeriodPrincipalPayment // 使用已计算好的本金偿还金额

			// 计算RoU资产的累计折旧(期初资产价值 - 期初账面价值)
			accumulatedDepreciation := result.InitialRoUAsset - result.PeriodRoUAssetStart

			// 计算期末累计折旧
			endAccumulatedDepreciation := accumulatedDepreciation + result.PeriodDepreciation

			// 主要财务指标表格
			// 第一列: 项目名称
			headers := []string{
				"项目",
				"使用权资产原值",
				"累计折旧",
				"使用权资产账面价值",
				"租赁负债",
				"本期折旧费用",
				"本期利息费用",
				"本期费用支出合计", // 新增：费用支出合计 = 折旧费用 + 利息费用
				"本期支付的租金",
				"其中：本金偿还",
				"其中：利息支付",
			}

			// 第二列: 期初余额
			startValues := []interface{}{
				"期初余额",
				result.InitialRoUAsset,
				accumulatedDepreciation,
				result.PeriodRoUAssetStart,
				result.PeriodLiabilityStart,
				"",
				"",
				"",
				"",
				"",
				"",
			}

			// 第三列: 期末余额
			endValues := []interface{}{
				"期末余额",
				result.InitialRoUAsset, // 原值不变
				endAccumulatedDepreciation,
				result.PeriodRoUAssetEnd,
				result.PeriodLiabilityEnd,
				"",
				"",
				"",
				"",
				"",
				"",
			}

			// 第四列: 本期发生额
			totalExpense := result.PeriodDepreciation + result.PeriodInterestExpense // 计算总费用支出
			periodValues := []interface{}{
				"本期发生额",
				"",
				result.PeriodDepreciation, // 本期新增的折旧
				result.PeriodRoUAssetEnd - result.PeriodRoUAssetStart,
				result.PeriodLiabilityEnd - result.PeriodLiabilityStart,
				result.PeriodDepreciation,
				result.PeriodInterestExpense,
				totalExpense, // 折旧费用 + 利息费用
				result.PeriodPayments,
				principalPayment,
				result.PeriodInterestExpense, // 利息支付等于利息费用
			}

			// 写入表格数据
			for i, header := range headers {
				row := 4 + i
				// 项目名称
				f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), header)
				// 期初值
				f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), startValues[i])
				// 期末值
				f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), endValues[i])
				// 本期发生额
				f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), periodValues[i])
			}

			// 设置列标题
			colHeaders := []string{"", "期初余额", "期末余额", "本期发生额"}
			for col, header := range colHeaders {
				cell := fmt.Sprintf("%c%d", 'A'+col, 3)
				f.SetCellValue(sheetName, cell, header)
			}

			// 格式化表头
			headerRange := fmt.Sprintf("A3:D3")
			f.SetCellStyle(sheetName, headerRange, headerRange, headerStyle)

			// 设置首列样式为粗体
			firstColStyle, _ := f.NewStyle(&excelize.Style{
				Font: &excelize.Font{Bold: true},
			})
			f.SetCellStyle(sheetName,
				fmt.Sprintf("A%d", 4),
				fmt.Sprintf("A%d", 4+len(headers)-1),
				firstColStyle)

			// 设置数字单元格格式
			numDataRange := fmt.Sprintf("B%d:D%d", 4, 4+len(headers)-1)
			f.SetCellStyle(sheetName, numDataRange, numDataRange, numStyle)

			// 更新基础行号,为后面的内容留出空间
			baseRow = 4 + len(headers) + 2 // 额外添加两行空行作为分隔
		}

		// Add lease details below the summary (if any)
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", baseRow), "Lease Details")
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", baseRow+1), "Lease ID:")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", baseRow+1), result.LeaseID)
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", baseRow+2), "Start Date:")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", baseRow+2), result.StartDate.Format("2006-01-02"))
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", baseRow+3), "End Date:")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", baseRow+3), result.EndDate.Format("2006-01-02"))
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", baseRow+4), "Payment Amount:")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", baseRow+4), result.PaymentAmount)
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", baseRow+5), "Payment Frequency:")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", baseRow+5), result.PaymentFrequency)
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", baseRow+6), "Discount Rate:")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", baseRow+6), result.DiscountRate)
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", baseRow+7), "Initial Lease Liability:")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", baseRow+7), result.InitialLiability)
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", baseRow+8), "Initial RoU Asset:")
		f.SetCellValue(sheetName, fmt.Sprintf("B%d", baseRow+8), result.InitialRoUAsset)

		// 调整后续表格的位置
		liabilityHeaderRow := baseRow + 10
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", liabilityHeaderRow), "Lease Liability Schedule")

		// Lease Liability headers
		liabHeaders := []string{"Period", "Date", "Opening Balance", "Payment",
			"Interest Expense", "Principal Repayment", "Closing Balance"}
		for i, header := range liabHeaders {
			cell := fmt.Sprintf("%c%d", 'A'+i, liabilityHeaderRow+1)
			f.SetCellValue(sheetName, cell, header)
		}

		// Format header row
		headerRange := fmt.Sprintf("A%d:%c%d", liabilityHeaderRow+1, 'A'+len(liabHeaders)-1, liabilityHeaderRow+1)
		f.SetCellStyle(sheetName, headerRange, headerRange, headerStyle)

		// Liability schedule data
		for i, entry := range result.LiabilitySchedule {
			row := i + liabilityHeaderRow + 2
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), entry.Period)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), entry.Date.Format("2006-01-02"))
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), entry.OpeningBalance)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), entry.Payment)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), entry.InterestExpense)
			f.SetCellValue(sheetName, fmt.Sprintf("F%d", row), entry.PrincipalRepayment)
			f.SetCellValue(sheetName, fmt.Sprintf("G%d", row), entry.ClosingBalance)
		}

		// Format liability schedule numbers
		liabDataRange := fmt.Sprintf("C%d:G%d", liabilityHeaderRow+2, liabilityHeaderRow+1+len(result.LiabilitySchedule))
		f.SetCellStyle(sheetName, liabDataRange, liabDataRange, numStyle)

		// Add RoU Asset Schedule
		firstRoURow := liabilityHeaderRow + len(result.LiabilitySchedule) + 3
		f.SetCellValue(sheetName, fmt.Sprintf("A%d", firstRoURow), "Right-of-Use Asset Schedule")

		// RoU Asset headers
		rouHeaders := []string{"Period", "Date", "Opening Balance", "Depreciation", "Closing Balance"}
		for i, header := range rouHeaders {
			cell := fmt.Sprintf("%c%d", 'A'+i, firstRoURow+1)
			f.SetCellValue(sheetName, cell, header)
		}

		// Format header row
		rouHeaderRange := fmt.Sprintf("A%d:%c%d", firstRoURow+1, 'A'+len(rouHeaders)-1, firstRoURow+1)
		f.SetCellStyle(sheetName, rouHeaderRange, rouHeaderRange, headerStyle)

		// RoU Asset schedule data
		for i, entry := range result.RoUAssetSchedule {
			row := i + firstRoURow + 2
			f.SetCellValue(sheetName, fmt.Sprintf("A%d", row), entry.Period)
			f.SetCellValue(sheetName, fmt.Sprintf("B%d", row), entry.Date.Format("2006-01-02"))
			f.SetCellValue(sheetName, fmt.Sprintf("C%d", row), entry.OpeningBalance)
			f.SetCellValue(sheetName, fmt.Sprintf("D%d", row), entry.Depreciation)
			f.SetCellValue(sheetName, fmt.Sprintf("E%d", row), entry.ClosingBalance)
		}

		// Format RoU asset schedule numbers
		rouDataRange := fmt.Sprintf("C%d:E%d", firstRoURow+2, firstRoURow+1+len(result.RoUAssetSchedule))
		f.SetCellStyle(sheetName, rouDataRange, rouDataRange, numStyle)

		// 删除原来的账期摘要部分(已经移到顶部了)

		// Adjust column widths
		for i := 0; i < 7; i++ {
			col := string(rune('A' + i))
			f.SetColWidth(sheetName, col, col, 15)
		}
	}

	// Set Summary as active sheet
	f.SetActiveSheet(0)

	// Save to buffer instead of file
	buffer, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}
