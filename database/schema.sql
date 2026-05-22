-- 1. สร้างตารางข้อมูลเล้าไก่ (coop)
CREATE TABLE IF NOT EXISTS coop (
    coop_id INT AUTO_INCREMENT PRIMARY KEY,
    date_adopt_animals DATE NOT NULL,
    amount INT NOT NULL,
    birthday DATE NOT NULL,
    note LONGTEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);

-- 2. สร้างตารางข้อมูลอุปกรณ์ (device)
CREATE TABLE IF NOT EXISTS device (
    device_id INT AUTO_INCREMENT PRIMARY KEY,
    coop_id INT,
    name VARCHAR(100) NOT NULL,
    device_type VARCHAR(50) NOT NULL,
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
    device_id INT,
    value DECIMAL(10,2) NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (device_id) REFERENCES device(device_id) ON DELETE CASCADE,
    INDEX idx_device_id (device_id),
    INDEX idx_timestamp (timestamp)
);

-- 4. สร้างตารางข้อมูลการเก็บไข่ (egg)
CREATE TABLE IF NOT EXISTS egg (
    egg_id INT AUTO_INCREMENT PRIMARY KEY,
    coop_id INT,
    date_collect_egg DATE,
    number_egg INT NOT NULL,
    note LONGTEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (coop_id) REFERENCES coop(coop_id) ON DELETE CASCADE,
    UNIQUE (coop_id, date_collect_egg),
    INDEX idx_coop_id (coop_id)
);

-- 5. สร้างตารางข้อมูลคลังอาหาร (foodstock)
CREATE TABLE IF NOT EXISTS foodstock (
    food_id INT AUTO_INCREMENT PRIMARY KEY,
    quantity_current DECIMAL(10,2) CHECK (quantity_current >= 0),
    min_quantity DECIMAL(10,2) DEFAULT 0 COMMENT 'ปริมาณขั้นต่ำ/เตือน',
    import_date DATE NOT NULL,
    expiry_date DATE NOT NULL,
    date_up TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_expiry_date (expiry_date),
    INDEX idx_min_quantity (min_quantity)
);

-- 6. สร้างตารางข้อมูลสุขภาพไก่ (health)
CREATE TABLE IF NOT EXISTS health (
    health_id INT AUTO_INCREMENT PRIMARY KEY,
    coop_id INT,
    healthy INT DEFAULT 0,
    poor_health INT DEFAULT 0,
    note LONGTEXT,
    record_date DATE NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (coop_id) REFERENCES coop(coop_id) ON DELETE CASCADE,
    INDEX idx_coop_id (coop_id),
    INDEX idx_record_date (record_date)
);

-- 7. สร้างตารางข้อมูลวัคซีน (vaccine)
CREATE TABLE IF NOT EXISTS vaccine (
    vaccine_id INT AUTO_INCREMENT PRIMARY KEY,
    coop_id INT,
    birthday DATE,
    name VARCHAR(50) NOT NULL,
    record_date DATE NOT NULL,
    method VARCHAR(100) NOT NULL,
    recommended_age VARCHAR(20) NOT NULL,
    note VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (coop_id) REFERENCES coop(coop_id) ON DELETE CASCADE,
    INDEX idx_coop_id (coop_id),
    INDEX idx_birthday (birthday)
);
