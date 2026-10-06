package models

import "time"

// bangkokLoc เป็น offset คงที่ UTC+7 (ไทยไม่มี DST เลยไม่ต้องพึ่ง IANA tzdata lookup
// ผ่าน time.LoadLocation ซึ่งอาจหาไม่เจอถ้า container ของ Render ไม่มีฐานข้อมูล
// timezone ติดตั้งมาให้)
var bangkokLoc = time.FixedZone("ICT", 7*60*60)

// DateKey แปลง time.Time เป็นวันที่ปฏิทินของไทย ("YYYY-MM-DD") เสมอ ไม่ว่า time.Time
// นั้นจะติด location อะไรมาก็ตาม - คอลัมน์ DATE/DATETIME ของ MySQL ไม่มีโซนเวลาในตัว
// ค่าที่ parseTime=True&loc=Local อ่านกลับมาจึงแค่ติด location ตาม time.Local ของ
// ตัว process Go เอง (เช่น UTC บน Render) ซึ่งไม่เกี่ยวกับ "วันที่ที่ควรจะเป็นตามเวลา
// ไทย" เลย - แปลงผ่านฟังก์ชันนี้ก่อน Format ทุกครั้งเพื่อให้ผลตรงกันไม่ว่าจะรันที่ไหน
// และยังช่วย "รักษา" แถวเก่าที่เคยถูกเขียนด้วยค่าที่แปลงเวลาเที่ยงคืนท้องถิ่นเป็น UTC
// จริงๆ มาก่อน (เช่น Date.toISOString() ของเบราว์เซอร์ ซึ่งเที่ยงคืนของไทยจะกลาย
// เป็น 17:00 ของวันก่อนหน้าใน UTC) ให้กลับมาโชว์วันที่ถูกต้องได้เองโดยไม่ต้องแก้ข้อมูล
func DateKey(t time.Time) string {
	return t.In(bangkokLoc).Format("2006-01-02")
}
