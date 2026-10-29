import "dotenv/config";
import request from "supertest";
import { describe, expect, it } from "vitest";
import { app } from "../src/app.js";

describe("GET /health", () => {
  it("reports the API and database as up", async () => {
    const res = await request(app).get("/health");
    expect(res.status).toBe(200);
    expect(res.body).toEqual({ status: "ok", db: "ok" });
  });
});

describe("unknown routes", () => {
  it("return a JSON 404", async () => {
    const res = await request(app).get("/nope");
    expect(res.status).toBe(404);
  });
});
