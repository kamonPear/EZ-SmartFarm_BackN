## Coop Management API

### Endpoints

#### 1. Create Coop (POST)
**Endpoint:** `POST /api/coops`

**Request Body:**
```json
{
  "date_adopt_animals": "2024-01-15T00:00:00Z",
  "amount": 50,
  "birthday": "2024-01-10T00:00:00Z",
  "note": "กลุ่มแรกของเล้าไก่"
}
```

**Required Fields:**
- `date_adopt_animals`: วันที่รับสัตว์เลี้ยง (datetime)
- `amount`: จำนวนตัว (integer, ≥1)
- `birthday`: วันเกิด (datetime)

**Optional Fields:**
- `note`: หมายเหตุ (string)

**Response (201 Created):**
```json
{
  "coop_id": 1,
  "date_adopt_animals": "2024-01-15T00:00:00Z",
  "amount": 50,
  "birthday": "2024-01-10T00:00:00Z",
  "note": "กลุ่มแรกของเล้าไก่",
  "devices": null,
  "eggs": null,
  "health": null,
  "vaccines": null
}
```

---

#### 2. Get All Coops (GET)
**Endpoint:** `GET /api/coops`

**Response (200 OK):**
```json
[
  {
    "coop_id": 1,
    "date_adopt_animals": "2024-01-15T00:00:00Z",
    "amount": 50,
    "birthday": "2024-01-10T00:00:00Z",
    "note": "กลุ่มแรกของเล้าไก่",
    "devices": [],
    "eggs": [],
    "health": [],
    "vaccines": []
  }
]
```

---

#### 3. Get Coop by ID (GET)
**Endpoint:** `GET /api/coops?id=1`

**Query Parameters:**
- `id`: Coop ID (required)

**Response (200 OK):**
```json
{
  "coop_id": 1,
  "date_adopt_animals": "2024-01-15T00:00:00Z",
  "amount": 50,
  "birthday": "2024-01-10T00:00:00Z",
  "note": "กลุ่มแรกของเล้าไก่",
  "devices": [],
  "eggs": [],
  "health": [],
  "vaccines": []
}
```

**Response (404 Not Found):**
```
Coop not found
```

---

#### 4. Update Coop (PUT)
**Endpoint:** `PUT /api/coops?id=1`

**Query Parameters:**
- `id`: Coop ID (required)

**Request Body (all fields optional):**
```json
{
  "date_adopt_animals": "2024-01-15T00:00:00Z",
  "amount": 55,
  "birthday": "2024-01-10T00:00:00Z",
  "note": "กลุ่มแรกของเล้าไก่ - อัปเดต"
}
```

**Response (200 OK):**
```json
{
  "coop_id": 1,
  "date_adopt_animals": "2024-01-15T00:00:00Z",
  "amount": 55,
  "birthday": "2024-01-10T00:00:00Z",
  "note": "กลุ่มแรกของเล้าไก่ - อัปเดต",
  "devices": [],
  "eggs": [],
  "health": [],
  "vaccines": []
}
```

---

#### 5. Delete Coop (DELETE)
**Endpoint:** `DELETE /api/coops?id=1`

**Query Parameters:**
- `id`: Coop ID (required)

**Response (200 OK):**
```json
{
  "message": "Coop deleted successfully"
}
```

**Notes:**
- ลบ Coop จะลบข้อมูลที่เกี่ยวข้องทั้งหมด (Devices, Eggs, Health, Vaccines)
- ใช้ Transaction เพื่อความปลอดภัย

---

## Folder Structure

```
EZ-SmartFarm_BackN/
├── models/
│   ├── models.go
│   ├── coop.go
│   ├── coop_request.go      ← Request/Response DTOs
│   ├── device.go
│   ├── sensor_log.go
│   ├── egg.go
│   ├── foodstock.go
│   ├── health.go
│   └── vaccine.go
├── database/
│   ├── db.go
│   ├── migration.go
│   └── coop_repository.go   ← CRUD operations for Coop
├── handlers/
│   ├── health.go
│   └── coop.go              ← API handlers for Coop
├── routes/
│   └── routes.go
├── main.go
└── .env
```

---

## Usage Example

### Using cURL

**Create Coop:**
```bash
curl -X POST http://localhost:8080/api/coops \
  -H "Content-Type: application/json" \
  -d '{
    "date_adopt_animals": "2024-01-15T00:00:00Z",
    "amount": 50,
    "birthday": "2024-01-10T00:00:00Z",
    "note": "First batch"
  }'
```

**Get All Coops:**
```bash
curl http://localhost:8080/api/coops
```

**Get Single Coop:**
```bash
curl http://localhost:8080/api/coops?id=1
```

**Update Coop:**
```bash
curl -X PUT http://localhost:8080/api/coops?id=1 \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 55,
    "note": "Updated batch"
  }'
```

**Delete Coop:**
```bash
curl -X DELETE http://localhost:8080/api/coops?id=1
```
