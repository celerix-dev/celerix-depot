import { v4 as uuidv4 } from 'uuid';

export const getClientID = (): string => {
  let clientID = localStorage.getItem('depot_client_id');
  if (!clientID) {
    clientID = uuidv4();
    localStorage.setItem('depot_client_id', clientID);
  }
  return clientID;
};

export const setClientID = (id: string) => {
  localStorage.setItem('depot_client_id', id);
};

export const getAdminSecret = (): string => {
  return localStorage.getItem('depot_admin_secret') || '';
};

export const setAdminSecret = (secret: string) => {
  localStorage.setItem('depot_admin_secret', secret);
};

export const clearAdminSecret = () => {
  localStorage.removeItem('depot_admin_secret');
};

export interface PersonaData {
  persona: string;
  name: string;
  recovery_code?: string;
  version?: string;
}

export const fetchPersona = async (): Promise<PersonaData> => {
  try {
    const response = await fetch('/api/persona', {
      headers: {
        'X-Client-ID': getClientID(),
        'X-Admin-Secret': getAdminSecret(),
      },
    });
    if (response.ok) {
      return await response.json();
    }
  } catch (error) {
    console.error('Error fetching persona:', error);
  }
  return { persona: 'client', name: '' };
};

export const updateClientName = async (name: string): Promise<{ success: boolean; recovery_code?: string }> => {
  try {
    const response = await fetch('/api/persona/name', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Client-ID': getClientID(),
        'X-Admin-Secret': getAdminSecret(),
      },
      body: JSON.stringify({ name }),
    });
    if (response.ok) {
      const data = await response.json();
      return { success: true, recovery_code: data.recovery_code };
    }
    return { success: false };
  } catch (error) {
    console.error('Error updating client name:', error);
    return { success: false };
  }
};

export const recoverPersona = async (code: string): Promise<{ success: boolean; persona?: string; name?: string }> => {
  try {
    const response = await fetch('/api/persona/recover', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
      },
      body: JSON.stringify({ code }),
    });
    if (response.ok) {
      const data = await response.json();
      if (data.persona === 'admin') {
        setAdminSecret(code);
      } else {
        setClientID(data.id);
        clearAdminSecret(); // Ensure we don't have a stale admin secret if we recovered a client
      }
      return { success: true, persona: data.persona, name: data.name };
    }
    return { success: false };
  } catch (error) {
    console.error('Error recovering persona:', error);
    return { success: false };
  }
};