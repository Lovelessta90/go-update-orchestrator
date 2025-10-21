// API Types matching Go structs

export interface Device {
    ID: string;
    Name: string;
    Address: string;
    Status: DeviceStatus;
    LastSeen: string | null;
    FirmwareVersion: string;
    Location: string;
    Metadata: Record<string, string>;
    CreatedAt: string;
    UpdatedAt: string;
}

export type DeviceStatus = 'online' | 'offline' | 'unknown';

export interface UpdateStatus {
    UpdateID: string;
    Status: UpdateStatusType;
    TotalDevices: number;
    Completed: number;
    Failed: number;
    InProgress: number;
    DeviceStatus: Record<string, DeviceUpdateStatus> | null;
    StartedAt: string;
    CompletedAt: string | null;
    EstimatedEnd: string | null;
}

export type UpdateStatusType = 'pending' | 'scheduled' | 'in_progress' | 'completed' | 'failed' | 'cancelled';

export interface DeviceUpdateStatus {
    Status: string;
    Error: string;
    StartedAt: string;
    CompletedAt: string | null;
}

export interface Stats {
    totalDevices: number;
    onlineDevices: number;
    activeUpdates: number;
}
