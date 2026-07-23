import { BrowserRouter, Routes, Route } from "react-router-dom";
import { CreatePage } from "./pages/CreatePage";
import { StatsPage } from "./pages/StatsPage";

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<CreatePage />} />
        <Route path="/stats/:code" element={<StatsPage />} />
      </Routes>
    </BrowserRouter>
  );
}
