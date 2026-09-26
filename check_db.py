from pymongo import MongoClient
import os

client = MongoClient("mongodb+srv://itsh4r08_db_user:r7hGECaBfQ0DAuks@src.qphvyxz.mongodb.net/?appName=src")
db = client.get_database("telegram_bot_db")
coll = db.get_collection("bot_settings")

doc = coll.find_one({"_id": "main_settings"})
if not doc:
    print("NO main_settings DOCUMENT FOUND!")
else:
    data = doc.get("data", {})
    users = data.get("users", {})
    batches = data.get("free_batches", {})
    print(f"Users found: {len(users)}")
    print(f"Batches found: {len(batches)}")
    print(f"Admin IDs: {data.get('admin_ids')}")
