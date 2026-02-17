import express from "express";

const app = express();

app.get("/api/health", (_req, res) => {
  res.json({ status: "ok" });
});

app.listen(3000, () => {
  console.log("Server running on port 3000");
});
