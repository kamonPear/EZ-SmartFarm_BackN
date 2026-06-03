package models

import (
	"time"
)

// VaccineAgeSchedule represents the vaccine schedule based on chicken age
type VaccineAgeSchedule struct {
	AgeInDaysMin   int    `json:"age_in_days_min"`
	AgeInDaysMax   int    `json:"age_in_days_max"`
	VaccineName    string `json:"vaccine_name"`
	Method         string `json:"method"`
	RecommendedAge string `json:"recommended_age"`
	Note           string `json:"note"`
}

// DeleteVaccineRequest represents a delete vaccine request
type DeleteVaccineRequest struct {
	VaccineID int `json:"vaccine_id"`
}

// GetVaccineScheduleByAge returns the vaccine schedule for a given age in days
// Returns vaccines that match the current age range, or the next vaccine to receive
func GetVaccineScheduleByAge(ageInDays int) []VaccineAgeSchedule {
	schedules := []VaccineAgeSchedule{
		{
			AgeInDaysMin:   3,
			AgeInDaysMax:   3,
			VaccineName:    "Marek's",
			Method:         "ฉีดใต้ผิวหนัง",
			RecommendedAge: "3 วัน",
			Note:           "โรคมาร์เก",
		},
		{
			AgeInDaysMin:   7,
			AgeInDaysMax:   10,
			VaccineName:    "Newcastle + IB",
			Method:         "หยอดตา หยอดจมูก ผสมน้ำ",
			RecommendedAge: "7 - 10 วัน",
			Note:           "โรคนิวคาสเซิล (Newcastle) + หลอดลมอักเสบ (IB)",
		},
		{
			AgeInDaysMin:   14,
			AgeInDaysMax:   14,
			VaccineName:    "Gumboro",
			Method:         "ผสมน้ำให้ดื่ม",
			RecommendedAge: "14 วัน",
			Note:           "ผสมน้ำให้ดื่ม",
		},
		{
			AgeInDaysMin:   21,
			AgeInDaysMax:   21,
			VaccineName:    "Gumboro (Pre-immunity 2)",
			Method:         "ผสมน้ำให้ดื่ม",
			RecommendedAge: "21 วัน",
			Note:           "ผสมน้ำให้ดื่ม",
		},
		{
			AgeInDaysMin:   28,
			AgeInDaysMax:   55,
			VaccineName:    "Fowl Pox",
			Method:         "แงว่หมัดที่ปีก",
			RecommendedAge: "4 - 8 สัปดาห์",
			Note:           "แงว่หมัดที่ปีก",
		},
		{
			AgeInDaysMin:   56,
			AgeInDaysMax:   84,
			VaccineName:    "Fowl Cholera",
			Method:         "ฉีดเข้ากล้ามเนื้อ",
			RecommendedAge: "8 - 12 สัปดาห์",
			Note:           "ฉีดเข้ากล้ามเนื้อ",
		},
	}

	// First, try to find vaccines that match current age
	var matchedSchedules []VaccineAgeSchedule
	for _, schedule := range schedules {
		if ageInDays >= schedule.AgeInDaysMin && ageInDays <= schedule.AgeInDaysMax {
			matchedSchedules = append(matchedSchedules, schedule)
		}
	}

	// If found vaccines for current age, return them
	if len(matchedSchedules) > 0 {
		return matchedSchedules
	}

	// If no match, find the next vaccine (lowest min age that is > current age)
	var nextVaccine *VaccineAgeSchedule
	for i := range schedules {
		if schedules[i].AgeInDaysMin > ageInDays {
			if nextVaccine == nil || schedules[i].AgeInDaysMin < nextVaccine.AgeInDaysMin {
				nextVaccine = &schedules[i]
			}
		}
	}

	if nextVaccine != nil {
		return []VaccineAgeSchedule{*nextVaccine}
	}

	// If age is beyond all schedules, return empty
	return []VaccineAgeSchedule{}
}

// CalculateChickenAge calculates the age of chicken in days from birthday
// Converts both birthday and current date to date-only format (no time component) for accurate calculation
func CalculateChickenAge(birthday time.Time) int {
	if birthday.IsZero() {
		return 0
	}

	// Get current time
	now := time.Now()

	// Normalize both dates to ignore time component (set to 00:00:00)
	birthDate := time.Date(birthday.Year(), birthday.Month(), birthday.Day(), 0, 0, 0, 0, time.UTC)
	todayDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)

	// Calculate the difference in days
	ageDuration := todayDate.Sub(birthDate)
	ageInDays := int(ageDuration.Hours() / 24)

	return ageInDays
}

// GetAllVaccineSchedules returns all vaccine schedules (fixed data)
func GetAllVaccineSchedules() []VaccineAgeSchedule {
	return []VaccineAgeSchedule{
		{
			AgeInDaysMin:   3,
			AgeInDaysMax:   3,
			VaccineName:    "Marek's",
			Method:         "ฉีดใต้ผิวหนัง",
			RecommendedAge: "3 วัน",
			Note:           "โรคมาร์เก",
		},
		{
			AgeInDaysMin:   7,
			AgeInDaysMax:   10,
			VaccineName:    "Newcastle + IB",
			Method:         "หยอดตา หยอดจมูก ผสมน้ำ",
			RecommendedAge: "7 - 10 วัน",
			Note:           "โรคนิวคาสเซิล (Newcastle) + หลอดลมอักเสบ (IB)",
		},
		{
			AgeInDaysMin:   14,
			AgeInDaysMax:   14,
			VaccineName:    "Gumboro",
			Method:         "ผสมน้ำให้ดื่ม",
			RecommendedAge: "14 วัน",
			Note:           "ผสมน้ำให้ดื่ม",
		},
		{
			AgeInDaysMin:   21,
			AgeInDaysMax:   21,
			VaccineName:    "Gumboro (Pre-immunity 2)",
			Method:         "ผสมน้ำให้ดื่ม",
			RecommendedAge: "21 วันรัน",
			Note:           "ผสมน้ำให้ดื่ม",
		},
		{
			AgeInDaysMin:   28,
			AgeInDaysMax:   56,
			VaccineName:    "Fowl Pox",
			Method:         "แงว่หมัดที่ปีก",
			RecommendedAge: "4 - 8 สัปดาห์",
			Note:           "แงว่หมัดที่ปีก",
		},
		{
			AgeInDaysMin:   56,
			AgeInDaysMax:   84,
			VaccineName:    "Fowl Cholera",
			Method:         "ฉีดเข้ากล้ามเนื้อ",
			RecommendedAge: "8 - 12 สัปดาห์",
			Note:           "ฉีดเข้ากล้ามเนื้อ",
		},
	}
}


