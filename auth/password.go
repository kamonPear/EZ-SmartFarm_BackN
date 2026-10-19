// =====================================================================================
// ไฟล์ auth/password.go - เข้ารหัสรหัสผ่านด้วย bcrypt
// =====================================================================================
//
// แพ็กเกจ auth รวมเรื่องความปลอดภัยไว้ 3 ไฟล์:
//  - password.go   : เข้ารหัส/ตรวจรหัสผ่าน (ไฟล์นี้)
//  - jwt.go        : ออกและตรวจ token หลัง login
//  - middleware.go : ยามตรวจสิทธิ์ก่อนเข้า API
//
// ทำไมห้ามเก็บรหัสผ่านตรง ๆ ในฐานข้อมูล: ถ้า DB หลุด ทุกคนจะเห็นรหัสผ่านทันที
// จึงเก็บเป็น "hash" ของ bcrypt ซึ่งย้อนกลับเป็นรหัสผ่านเดิมไม่ได้
//
// ทำไมเลือก bcrypt แทน MD5/SHA256:
//  - bcrypt ตั้งใจให้ช้า (ปรับความยากได้ด้วย cost) คนร้ายลองเดารหัสรัว ๆ ได้ยากมาก
//  - ใส่ "salt" สุ่มให้อัตโนมัติ รหัสผ่านเดียวกันจึงได้ hash ต่างกันทุกครั้ง ใช้ตารางเดารหัสสำเร็จรูปไม่ได้
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword แปลงรหัสผ่านธรรมดาเป็น hash ของ bcrypt ก่อนบันทึกลงตาราง users
// ใช้ bcrypt.DefaultCost (ระดับความยาก 10) ซึ่งเป็นค่าที่ไลบรารีแนะนำ สมดุลระหว่างความปลอดภัยกับความเร็ว
//
// ถูกเรียกจาก:
//  - handlers/auth.go -> RegisterHandler ตอนผู้ดูแลสร้างบัญชีใหม่
//  - database/migration.go -> ตอนสร้างบัญชีตั้งต้น (seed) ให้ระบบ
func HashPassword(plain string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashed), nil
}

// CheckPassword ตรวจว่ารหัสผ่านที่ผู้ใช้กรอก (plain) ตรงกับ hash ที่เก็บไว้หรือไม่
// คืน true ถ้าตรง
//
// ทำงานอย่างไร: bcrypt ดึง salt และ cost ที่ฝังอยู่ใน hash ออกมา แล้ว hash รหัสที่กรอกด้วยค่าเดียวกัน
// จากนั้นเทียบผล (เราไม่ต้องถอดรหัส เพราะถอดไม่ได้อยู่แล้ว)
// ถูกเรียกจาก: handlers/auth.go -> LoginHandler ก่อนออก JWT ให้ผู้ใช้
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
