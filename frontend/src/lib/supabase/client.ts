import { createBrowserClient } from '@supabase/ssr';
import { Database } from '@/types/database.types';

export function createClient() {
  const supabaseUrl = process.env.NEXT_PUBLIC_SUPABASE_URL || 'https://lvsngrrdzjhbawhcuzqz.supabase.co';
  const supabaseAnonKey = process.env.NEXT_PUBLIC_SUPABASE_ANON_KEY || 'eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJzdXBhYmFzZSIsInJlZiI6Imx2c25ncnJkempoYmF3aGN1enF6Iiwicm9sZSI6ImFub24iLCJpYXQiOjE3ODg1MzAxMzEsImV4cCI6MjEwNDEwNjEzMX0.yilfkY7aUMr4j5Q0oj2oH8kGH4fIelCNZKDjo_VOfls';

  return createBrowserClient<Database>(supabaseUrl, supabaseAnonKey);
}

// Browser singleton client instance for convenient client-side access
let browserClient: ReturnType<typeof createClient> | null = null;

export function getSupabaseBrowserClient() {
  if (!browserClient) {
    browserClient = createClient();
  }
  return browserClient;
}
