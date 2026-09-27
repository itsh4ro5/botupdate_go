import React from 'react';
import { Plus, Play, Pause, Trash2 } from 'lucide-react';

export const Scheduler: React.FC = () => {
  return (
    <div className="fade-in" style={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <div className="page-header flex justify-between items-center">
        <div>
          <h1>Task Scheduler</h1>
          <p className="text-muted text-sm mt-1">Manage background jobs and automated tasks</p>
        </div>
        <button className="btn btn-primary flex items-center gap-2">
          <Plus size={16} /> New Task
        </button>
      </div>

      <div className="glass-panel" style={{ flex: 1 }}>
        <div className="table-responsive">
          <table className="table w-full">
            <thead>
              <tr>
                <th className="text-left p-4">Task Name</th>
                <th className="text-left p-4">Schedule (Cron)</th>
                <th className="text-left p-4">Next Run</th>
                <th className="text-left p-4">Status</th>
                <th className="text-right p-4">Actions</th>
              </tr>
            </thead>
            <tbody>
              <tr className="border-t border-white/5">
                <td className="p-4">
                  <div className="font-semibold">Auto-Backup DB</div>
                  <div className="text-xs text-muted">Dumps MongoDB to local disk</div>
                </td>
                <td className="p-4">
                  <div className="badge badge-outline">0 0 * * *</div>
                </td>
                <td className="p-4 text-sm">Today, 12:00 AM</td>
                <td className="p-4">
                  <span className="badge badge-success">Active</span>
                </td>
                <td className="p-4">
                  <div className="flex justify-end gap-2">
                    <button className="btn-icon text-muted hover:text-white" title="Pause">
                      <Pause size={18} />
                    </button>
                    <button className="btn-icon text-muted hover:text-error" title="Delete">
                      <Trash2 size={18} />
                    </button>
                  </div>
                </td>
              </tr>
              
              <tr className="border-t border-white/5">
                <td className="p-4">
                  <div className="font-semibold">Clear Expired Sessions</div>
                  <div className="text-xs text-muted">Removes old web sessions</div>
                </td>
                <td className="p-4">
                  <div className="badge badge-outline">0 * * * *</div>
                </td>
                <td className="p-4 text-sm">In 20 mins</td>
                <td className="p-4">
                  <span className="badge badge-success">Active</span>
                </td>
                <td className="p-4">
                  <div className="flex justify-end gap-2">
                    <button className="btn-icon text-muted hover:text-white" title="Pause">
                      <Pause size={18} />
                    </button>
                    <button className="btn-icon text-muted hover:text-error" title="Delete">
                      <Trash2 size={18} />
                    </button>
                  </div>
                </td>
              </tr>
              
              <tr className="border-t border-white/5 opacity-50">
                <td className="p-4">
                  <div className="font-semibold">Weekly Report</div>
                  <div className="text-xs text-muted">Generates PDF report</div>
                </td>
                <td className="p-4">
                  <div className="badge badge-outline">0 0 * * 0</div>
                </td>
                <td className="p-4 text-sm">-</td>
                <td className="p-4">
                  <span className="badge badge-warning">Paused</span>
                </td>
                <td className="p-4">
                  <div className="flex justify-end gap-2">
                    <button className="btn-icon text-muted hover:text-white" title="Resume">
                      <Play size={18} />
                    </button>
                    <button className="btn-icon text-muted hover:text-error" title="Delete">
                      <Trash2 size={18} />
                    </button>
                  </div>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  );
};
