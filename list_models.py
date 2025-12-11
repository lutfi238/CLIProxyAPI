#!/usr/bin/env python3
import requests

API_URL = "http://127.0.0.1:8317/v1/models"
API_KEY = "8a8aJFdbFTwe5WrickW52Qa2OoFz1fRfkY0Bmh1DoJDs2Klnvh"

resp = requests.get(API_URL, headers={"Authorization": f"Bearer {API_KEY}"})
models = resp.json()["data"]

print("\nAvailable models:\n")
for m in sorted(models, key=lambda x: x["id"]):
    print(m["id"])
print(f"\nTotal: {len(models)} models")
