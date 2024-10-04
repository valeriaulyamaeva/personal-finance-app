package database_test

import (
	"context"
	"github.com/joho/godotenv"
	"github.com/valeriaulyamaeva/personal-finance-app/internal/database"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
	"testing"
)

func TestCreateReport(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	report := &models.Report{
		UserID:     1,
		ReportData: "Monthly financial report",
	}

	err = database.CreateReport(conn, report)
	if err != nil {
		t.Fatalf("ошибка создания отчета: %v", err)
	}

	t.Logf("ID отчета после создания: %d", report.ID)

	createdReport, err := database.GetReportByID(conn, report.ID)
	if err != nil {
		t.Fatalf("ошибка получения отчета по ID: %v", err)
	}

	if createdReport.UserID != report.UserID || createdReport.ReportData != report.ReportData {
		t.Errorf("данные отчета не совпадают: получили %+v, хотели %+v", createdReport, report)
	}
}

func TestUpdateReport(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	report := &models.Report{
		UserID:     1,
		ReportData: "Initial report data",
	}
	err = database.CreateReport(conn, report)
	if err != nil {
		t.Fatalf("ошибка создания отчета: %v", err)
	}

	// Обновляем данные отчета
	report.ReportData = "Updated report data"
	err = database.UpdateReport(conn, report)
	if err != nil {
		t.Fatalf("ошибка обновления отчета: %v", err)
	}

	// Проверяем обновление
	updatedReport, err := database.GetReportByID(conn, report.ID)
	if err != nil {
		t.Fatalf("не смогли получить обновленный отчет по ID: %v", err)
	}

	if updatedReport.ReportData != report.ReportData {
		t.Errorf("данные отчета не совпадают после обновления: получили %+v, хотели %+v", updatedReport, report)
	}
}

func TestDeleteReport(t *testing.T) {
	if err := godotenv.Load(); err != nil {
		t.Fatalf("ошибка загрузки .env: %v", err)
	}
	conn, err := database.ConnectDB()
	if err != nil {
		t.Fatalf("ошибка подключения к БД: %v", err)
	}
	defer conn.Close(context.Background())

	report := &models.Report{
		UserID:     1,
		ReportData: "Report to delete",
	}
	err = database.CreateReport(conn, report)
	if err != nil {
		t.Fatalf("ошибка создания отчета: %v", err)
	}

	err = database.DeleteReport(conn, report.ID)
	if err != nil {
		t.Fatalf("ошибка удаления отчета: %v", err)
	}

	// Проверяем, что отчет удален
	_, err = database.GetReportByID(conn, report.ID)
	if err == nil {
		t.Errorf("ошибка удаления отчета по ID, отчет все еще существует")
	}
}
