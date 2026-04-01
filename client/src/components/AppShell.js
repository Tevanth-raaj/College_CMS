import React, { useCallback, useState } from "react";
import { Outlet } from "react-router-dom";
import Sidebar from "./Sidebar";

const AppShell = () => {
  const [isSidebarExpanded, setIsSidebarExpanded] = useState(false);

  const handleSidebarExpandedChange = useCallback((expanded) => {
    setIsSidebarExpanded(expanded);
  }, []);

  return (
    <div className="min-h-screen bg-background flex overflow-x-hidden">
      <Sidebar onExpandedChange={handleSidebarExpandedChange} />
      <div
        className={`flex-1 min-w-0 transition-all duration-300 ${
          isSidebarExpanded ? "ml-64" : "ml-20"
        }`}
      >
        <Outlet />
      </div>
    </div>
  );
};

export default AppShell;
