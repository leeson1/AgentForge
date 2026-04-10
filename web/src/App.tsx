import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom';
import { AppLayout } from './components/AppLayout';

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<AppLayout />}>
          <Route index element={<Navigate to="/tasks" replace />} />
          <Route path="tasks" element={null} />
          <Route path="tasks/:taskId" element={null} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
