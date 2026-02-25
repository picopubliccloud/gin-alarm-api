package utils

import (
	"bytes"
	"fmt"
	"reflect"

	"github.com/xuri/excelize/v2"
)

func GenerateExcel(data interface{}) (*bytes.Buffer, error) {

	// Check if data is a slice
	value := reflect.ValueOf(data)
	if value.Kind() != reflect.Slice {
		return nil, fmt.Errorf("Data must be a slice!")
	}

	// Check if slice is empty or not:
	if value.Len() == 0 {
		return nil, fmt.Errorf("Slice is empty!")
	}

	// Get the data type (struct type):
	dataType := value.Index(0).Type()
	if dataType.Kind() != reflect.Struct {
		return nil, fmt.Errorf("Slice element must be structs!")
	}

	// Create excel file:
	file := excelize.NewFile()
	sheetName := "Sheet1"
	index := 0

	// Write headers (only exported fields):
	colCount := 0
	for i := 0; i < dataType.NumField(); i++ {
		field := dataType.Field(i) // field name at i pos as string;
		if !field.IsExported() {
			continue // skips private fileds (starts with loswer case)
		}
		cell, _ := excelize.CoordinatesToCellName(colCount+1, 1) // 1 = A, 2 = B... 26 = z, 27 = AA & second param = 1 is for A1, B1... (first row)
		file.SetCellValue(sheetName, cell, field.Name)           // so headers are written in row 1.
		colCount++
	}

	for rowIndex := 0; rowIndex < value.Len(); rowIndex++ {
		rowValue := value.Index(rowIndex)
		colCount := 0
		for i := 0; i < dataType.NumField(); i++ {
			field := dataType.Field(i)
			if !field.IsExported() {
				continue
			}

			var fieldValue interface{}
			f := rowValue.Field(i)
			if f.Kind() == reflect.Ptr {
				if f.IsNil() {
					fieldValue = ""
				} else {
					fieldValue = f.Elem().Interface() // elem returns the value, the pointer points to.
				}
			} else {
				fieldValue = f.Interface()
			}

			cell, _ := excelize.CoordinatesToCellName(colCount+1, rowIndex+2)
			file.SetCellValue(sheetName, cell, fieldValue)
			colCount++
		}
	}

	file.SetActiveSheet(index) // sheet that opens by default after opening a file

	//  write to memory buffer
	buffer, err := file.WriteToBuffer()
	if err != nil {
		return nil, err
	}

	return buffer, nil
}
