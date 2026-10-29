import express, { type ErrorRequestHandler } from "express";
import cors from "cors";
import { prisma } from "./db.js";

export const app = express();

app.use(cors());
app.use(express.json({ limit: "200kb" }));

// Liveness plus a database round trip, so a broken DB connection shows up here.
app.get("/health", async (_req, res) => {
  try {
    await prisma.$queryRaw`SELECT 1`;
    res.json({ status: "ok", db: "ok" });
  } catch {
    res.status(503).json({ status: "degraded", db: "unreachable" });
  }
});

// Auth (Day 3) and job/application routes (Day 4) mount here.

app.use((_req, res) => {
  res.status(404).json({ error: "Not found" });
});

const errorHandler: ErrorRequestHandler = (err, _req, res, _next) => {
  console.error(err);
  res.status(500).json({ error: "Internal server error" });
};
app.use(errorHandler);
