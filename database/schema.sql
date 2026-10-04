-- ⚠️ ไฟล์นี้เป็นเอกสารอ้างอิงโครงสร้างฐานข้อมูลปัจจุบัน (ตรงกับ GORM models + live DB
-- ที่ตรวจสอบจริงแล้ว) ไม่ใช่สคริปต์ตั้งค่าแรกเริ่ม - ตัว backend เองสร้าง/ปรับตารางให้
-- อัตโนมัติผ่าน database.MigrateModels() (GORM AutoMigrate) ทุกครั้งที่ start ไม่ต้องรัน
-- ไฟล์นี้ด้วยมือ เว้นแต่ตั้งใจจะสร้างฐานข้อมูลเปล่าใหม่แยกต่างหาก
--
-- ❌ ห้ามรัน DROP DATABASE / DROP TABLE กับฐานข้อมูลที่ใช้งานจริงเด็ดขาด จะลบข้อมูล
-- ทั้งหมดถาวร กู้คืนไม่ได้ - ทุกคำสั่งด้านล่างใช้ CREATE TABLE IF NOT EXISTS เท่านั้น
-- ปลอดภัยที่จะรันซ้ำกับฐานข้อมูลที่มีอยู่แล้ว (จะข้ามตารางที่มีอยู่แล้วเฉย ๆ)

-- 0. ตารางบัญชีผู้ใช้ (User) - ไม่มี role, ทุกบัญชีเท่ากันหมด แต่ละบัญชีมีข้อมูลฟาร์ม
-- ของตัวเองแยกขาดจากกันโดยสิ้นเชิงผ่าน user_id ในทุกตารางด้านล่าง
CREATE TABLE IF NOT EXISTS `User` (
    id_User INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(100) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 1. สร้างตารางข้อมูลเล้าไก่ (coop)
CREATE TABLE IF NOT EXISTS coop (
    coop_id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    name_coop VARCHAR(100) UNIQUE,
    date_adopt_animals DATE NOT NULL,
    amount INT NOT NULL,
    birthday DATE NOT NULL,
    note LONGTEXT,
    pos_x DOUBLE NULL,
    pos_y DOUBLE NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES `User`(id_User) ON DELETE CASCADE,
    INDEX idx_user_id (user_id)
);

-- 2. สร้างตารางข้อมูลอุปกรณ์ (device)
CREATE TABLE IF NOT EXISTS device (
    device_id INT AUTO_INCREMENT PRIMARY KEY,
    coop_id INT NOT NULL,
    name_coop VARCHAR(100),
    slot_index INT NOT NULL,
    name VARCHAR(100),
    icon VARCHAR(255),
    device_type VARCHAR(50),
    current_status VARCHAR(20) DEFAULT 'Offline',
    last_update TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (coop_id) REFERENCES coop(coop_id) ON DELETE CASCADE,
    INDEX idx_coop_id (coop_id)
);

-- 3. สร้างตารางบันทึกข้อมูลเซนเซอร์ (sensor_log)
CREATE TABLE IF NOT EXISTS sensor_log (
    log_id INT AUTO_INCREMENT PRIMARY KEY,
    coop_id INT,
    device_id INT,
    name VARCHAR(100),
    value DECIMAL(10,2) NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES device(device_id) ON DELETE CASCADE,
    INDEX idx_device_id (device_id),
    INDEX idx_timestamp (timestamp)
);

-- 4. สร้างตารางข้อมูลการเก็บไข่ (egg)
CREATE TABLE IF NOT EXISTS egg (
    egg_id INT AUTO_INCREMENT PRIMARY KEY,
    coop_id INT NOT NULL,
    name_coop VARCHAR(100),
    date_collect_egg DATE,
    number_egg INT NOT NULL,
    note LONGTEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (coop_id) REFERENCES coop(coop_id) ON DELETE CASCADE,
    UNIQUE (coop_id, date_collect_egg),
    INDEX idx_coop_id (coop_id)
);

-- 5. สร้างตารางบันทึกการนำเข้าอาหารแต่ละล็อต (importfood) - แยกต่อ user
CREATE TABLE IF NOT EXISTS importfood (
    lot_id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    food_type VARCHAR(20) NOT NULL,
    import_volume INT NOT NULL,
    import_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expiry_date DATE NOT NULL,
    FOREIGN KEY (user_id) REFERENCES `User`(id_User) ON DELETE CASCADE,
    INDEX idx_user_id (user_id)
);

-- 6. สร้างตารางยอดรวมอาหารคงเหลือปัจจุบัน (foodstock) - 1 แถวต่อ (user_id, food_type)
-- อัปเดตอัตโนมัติทุกครั้งที่มีการเพิ่มแถวใน importfood
CREATE TABLE IF NOT EXISTS foodstock (
    food_id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    food_type VARCHAR(20) NOT NULL,
    quantity_current DECIMAL(10,2) CHECK (quantity_current >= 0),
    date_up DATE NOT NULL,
    FOREIGN KEY (user_id) REFERENCES `User`(id_User) ON DELETE CASCADE,
    UNIQUE uq_foodstock_type_user (user_id, food_type)
);

-- 6b. สร้างตารางประวัติการแจกจ่ายอาหารตอนตัดสต็อก (food_distribution) - 1 แถวต่อคอก
-- ต่อรอบตัด เก็บถาวรไว้เป็นประวัติ ไม่เคยเขียนทับ
CREATE TABLE IF NOT EXISTS food_distribution (
    distribution_id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    food_type VARCHAR(20) NOT NULL,
    coop_id INT NOT NULL,
    kg_given DECIMAL(10,2) NOT NULL,
    distributed_at TIMESTAMP NOT NULL,
    FOREIGN KEY (user_id) REFERENCES `User`(id_User) ON DELETE CASCADE,
    FOREIGN KEY (coop_id) REFERENCES coop(coop_id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_food_type (food_type)
);

-- 7. สร้างตารางข้อมูลสุขภาพไก่ (health)
CREATE TABLE IF NOT EXISTS health (
    health_id INT AUTO_INCREMENT PRIMARY KEY,
    coop_id INT NOT NULL,
    number_healthy INT DEFAULT 0,
    number_poor_health INT DEFAULT 0,
    note LONGTEXT,
    `date` DATE NOT NULL,
    FOREIGN KEY (coop_id) REFERENCES coop(coop_id) ON DELETE CASCADE,
    INDEX idx_coop_id (coop_id),
    INDEX idx_date (`date`)
);

-- 7b. สร้างตารางนัดตรวจสุขภาพที่เจ้าของฟาร์มตั้งเอง (health_appointment) - แยกจาก
-- การแจ้งเตือนที่คำนวณอัตโนมัติจากกำหนดวัคซีน
CREATE TABLE IF NOT EXISTS health_appointment (
    appointment_id INT AUTO_INCREMENT PRIMARY KEY,
    coop_id INT NOT NULL,
    appointment_date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (coop_id) REFERENCES coop(coop_id) ON DELETE CASCADE,
    INDEX idx_coop_id (coop_id),
    INDEX idx_appointment_date (appointment_date)
);

-- 8. สร้างตารางประเภทยา/วัคซีน (vaccine) - เก็บแค่ "เกณฑ์" (ชื่อ, วิธีให้, ช่วงอายุ
-- ที่ควรให้) ไม่ผูกกับคอกไหนทั้งนั้น และไม่ผูกกับ user คนไหนด้วย - ใช้ร่วมกันได้ทุก
-- user ทุกคอก (ต่างจากตารางอื่นทั้งหมดในไฟล์นี้)
CREATE TABLE IF NOT EXISTS vaccine (
    vaccine_id INT AUTO_INCREMENT PRIMARY KEY,
    name_vaccine VARCHAR(50) NOT NULL,
    method VARCHAR(100) NOT NULL,
    min_age_days INT NOT NULL,
    max_age_days INT NOT NULL,
    note VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 8b. สร้างตารางประวัติการให้วัคซีนจริง (vaccine_history) - แยกจาก vaccine ด้านบน
-- ตารางนี้ทุกแถวคือเหตุการณ์ "ให้จริง" กับคอกใดคอกหนึ่งแล้วเสมอ (ผูกกับ user ผ่าน coop)
CREATE TABLE IF NOT EXISTS vaccine_history (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    coop_id INT NOT NULL,
    name_coop VARCHAR(100),
    birthday DATE,
    name_vaccine VARCHAR(50) NOT NULL,
    method VARCHAR(100) NOT NULL,
    record_date DATE NOT NULL,
    recommended_age VARCHAR(20),
    note VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (coop_id) REFERENCES coop(coop_id) ON DELETE CASCADE,
    INDEX idx_coop_id (coop_id),
    INDEX idx_name_coop (name_coop)
);

-- 9. สร้างตารางค่ามาตรฐานของฟาร์ม (farm_threshold) - 1 แถวต่อ user เท่านั้น ใช้ร่วมกัน
-- ทุกคอกของ user นั้น แถวจะถูกสร้างก็ต่อเมื่อ user กดตั้งค่าเองครั้งแรกเท่านั้น (ไม่ใช่
-- auto-create ตอนเปิดหน้า Dashboard - เคยเป็นบั๊กที่ user ใหม่เห็นค่า default ของคนอื่น)
CREATE TABLE IF NOT EXISTS farm_threshold (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    temperature DECIMAL(5,2) NOT NULL,
    ammonia DECIMAL(5,2) NOT NULL,
    FOREIGN KEY (user_id) REFERENCES `User`(id_User) ON DELETE CASCADE,
    UNIQUE uq_farm_threshold_user (user_id)
);

-- 10. สร้างตารางผังรูปร่างฟาร์ม (farm_layout) - 1 แถวต่อ user เช่นกัน สร้างเมื่อ user
-- เลือกรูปร่างเองครั้งแรกเท่านั้น (หลักการเดียวกับ farm_threshold ด้านบน)
CREATE TABLE IF NOT EXISTS farm_layout (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,
    shape VARCHAR(20),
    FOREIGN KEY (user_id) REFERENCES `User`(id_User) ON DELETE CASCADE,
    UNIQUE uq_farm_layout_user (user_id)
);
