-- 1. สร้างตารางข้อมูลเล้าไก่ (coop)
CREATE TABLE IF NOT EXISTS coop (
    coop_id INT AUTO_INCREMENT PRIMARY KEY,
    name_coop VARCHAR(100) UNIQUE,
    date_adopt_animals DATE NOT NULL,
    amount INT NOT NULL,
    birthday DATE NOT NULL,
    note LONGTEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE (birthday)
);

-- 2. สร้างตารางข้อมูลอุปกรณ์ (device)
CREATE TABLE IF NOT EXISTS device (
    device_id INT AUTO_INCREMENT PRIMARY KEY,
    coop_id INT,
    name_coop VARCHAR(100),
    name VARCHAR(100) NOT NULL UNIQUE,
    device_type VARCHAR(50) NOT NULL,
    current_status VARCHAR(20) DEFAULT 'Offline',
    last_update TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (coop_id) REFERENCES coop(coop_id) ON DELETE CASCADE,
    FOREIGN KEY (name_coop) REFERENCES coop(name_coop),
    INDEX idx_coop_id (coop_id)
);

-- 3. สร้างตารางบันทึกข้อมูลเซนเซอร์ (sensor_log)
CREATE TABLE IF NOT EXISTS sensor_log (
    log_id INT AUTO_INCREMENT PRIMARY KEY,
    device_id INT,
    name VARCHAR(100),
    value DECIMAL(10,2) NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES device(device_id) ON DELETE CASCADE,
    FOREIGN KEY (name) REFERENCES device(name),
    INDEX idx_device_id (device_id),
    INDEX idx_timestamp (timestamp)
);

-- 4. สร้างตารางข้อมูลการเก็บไข่ (egg)
CREATE TABLE IF NOT EXISTS egg (
    egg_id INT AUTO_INCREMENT PRIMARY KEY,
    coop_id INT,
    name_coop VARCHAR(100),
    date_collect_egg DATE,
    number_egg INT NOT NULL,
    note LONGTEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (coop_id) REFERENCES coop(coop_id) ON DELETE CASCADE,
    FOREIGN KEY (name_coop) REFERENCES coop(name_coop),
    UNIQUE (coop_id, date_collect_egg),
    INDEX idx_coop_id (coop_id)
);

-- 5. สร้างตารางบันทึกการนำเข้าอาหารแต่ละล็อต (importfood)
CREATE TABLE IF NOT EXISTS importfood (
    lot_id INT AUTO_INCREMENT PRIMARY KEY,
    import_volume INT NOT NULL,
    import_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expiry_date DATE NOT NULL
);

-- 6. สร้างตารางยอดรวมอาหารคงเหลือปัจจุบัน (foodstock) - อัปเดตอัตโนมัติทุกครั้งที่มีการเพิ่มแถวใน importfood
CREATE TABLE IF NOT EXISTS foodstock (
    food_id INT AUTO_INCREMENT PRIMARY KEY,
    quantity_current DECIMAL(10,2) CHECK (quantity_current >= 0),
    date_up DATE NOT NULL
);

-- 7. สร้างตารางข้อมูลสุขภาพไก่ (health)
CREATE TABLE IF NOT EXISTS health (
    health_id INT AUTO_INCREMENT PRIMARY KEY,
    coop_id INT,
    name_coop VARCHAR(100),
    healthy INT DEFAULT 0,
    poor_health INT DEFAULT 0,
    note LONGTEXT,
    record_date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (coop_id) REFERENCES coop(coop_id) ON DELETE CASCADE,
    FOREIGN KEY (name_coop) REFERENCES coop(name_coop),
    INDEX idx_coop_id (coop_id),
    INDEX idx_record_date (record_date)
);

-- 8. สร้างตารางประเภทยา/วัคซีน (vaccine) - เก็บแค่ "เกณฑ์" (ชื่อ, วิธีให้, ช่วงอายุ
-- ที่ควรให้) ไม่ผูกกับคอกไหนทั้งนั้น ใช้ร่วมกันได้กับทุกคอก
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
-- ตารางนี้ทุกแถวคือเหตุการณ์ "ให้จริง" กับคอกใดคอกหนึ่งแล้วเสมอ
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
