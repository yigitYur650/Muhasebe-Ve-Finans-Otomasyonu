export type Json =
  | string
  | number
  | boolean
  | null
  | { [key: string]: Json | undefined }
  | Json[]

export type Database = {
  public: {
    Tables: {
      tenants: {
        Row: {
          id: string
          name: string
          created_at: string
        }
        Insert: {
          id?: string
          name: string
          created_at?: string
        }
        Update: {
          id?: string
          name?: string
          created_at?: string
        }
        Relationships: []
      }
      tenant_members: {
        Row: {
          id: string
          tenant_id: string
          user_id: string
          role: 'admin' | 'muhasebeci' | 'standart'
          created_at: string
        }
        Insert: {
          id?: string
          tenant_id: string
          user_id: string
          role: 'admin' | 'muhasebeci' | 'standart'
          created_at?: string
        }
        Update: {
          id?: string
          tenant_id?: string
          user_id?: string
          role?: 'admin' | 'muhasebeci' | 'standart'
          created_at?: string
        }
        Relationships: [
          {
            foreignKeyName: "tenant_members_tenant_id_fkey"
            columns: ["tenant_id"]
            isOneToOne: false
            referencedRelation: "tenants"
            referencedColumns: ["id"]
          }
        ]
      }
      periods: {
        Row: {
          id: string
          tenant_id: string
          label: string
          starting_balance: number
          status: 'open' | 'locked'
          opened_at: string
          locked_at: string | null
        }
        Insert: {
          id?: string
          tenant_id: string
          label: string
          starting_balance?: number
          status?: 'open' | 'locked'
          opened_at?: string
          locked_at?: string | null
        }
        Update: {
          id?: string
          tenant_id?: string
          label?: string
          starting_balance?: number
          status?: 'open' | 'locked'
          opened_at?: string
          locked_at?: string | null
        }
        Relationships: [
          {
            foreignKeyName: "periods_tenant_id_fkey"
            columns: ["tenant_id"]
            isOneToOne: false
            referencedRelation: "tenants"
            referencedColumns: ["id"]
          }
        ]
      }
      transactions: {
        Row: {
          id: string
          tenant_id: string
          period_id: string
          direction: 'in' | 'out'
          channel:
            | 'eft'
            | 'pos'
            | 'nakit'
            | 'kredi'
            | 'kira'
            | 'maas_banka'
            | 'maas_elden'
            | 'kredi_karti'
            | 'kartus'
            | 'yemek'
            | 'yakit'
            | 'diger'
          amount: number
          description: string | null
          created_by: string
          created_at: string
          reversed_by: string | null
        }
        Insert: {
          id?: string
          tenant_id: string
          period_id: string
          direction: 'in' | 'out'
          channel:
            | 'eft'
            | 'pos'
            | 'nakit'
            | 'kredi'
            | 'kira'
            | 'maas_banka'
            | 'maas_elden'
            | 'kredi_karti'
            | 'kartus'
            | 'yemek'
            | 'yakit'
            | 'diger'
          amount: number
          description?: string | null
          created_by: string
          created_at?: string
          reversed_by?: string | null
        }
        Update: {
          id?: string
          tenant_id?: string
          period_id?: string
          direction?: 'in' | 'out'
          channel?:
            | 'eft'
            | 'pos'
            | 'nakit'
            | 'kredi'
            | 'kira'
            | 'maas_banka'
            | 'maas_elden'
            | 'kredi_karti'
            | 'kartus'
            | 'yemek'
            | 'yakit'
            | 'diger'
          amount?: number
          description?: string | null
          created_by?: string
          created_at?: string
          reversed_by?: string | null
        }
        Relationships: [
          {
            foreignKeyName: "transactions_period_id_fkey"
            columns: ["period_id"]
            isOneToOne: false
            referencedRelation: "periods"
            referencedColumns: ["id"]
          },
          {
            foreignKeyName: "transactions_tenant_id_fkey"
            columns: ["tenant_id"]
            isOneToOne: false
            referencedRelation: "tenants"
            referencedColumns: ["id"]
          }
        ]
      }
      idempotency_keys: {
        Row: {
          key: string
          tenant_id: string
          response_body: Json | null
          response_status: number | null
          created_at: string
        }
        Insert: {
          key: string
          tenant_id: string
          response_body?: Json | null
          response_status?: number | null
          created_at?: string
        }
        Update: {
          key?: string
          tenant_id?: string
          response_body?: Json | null
          response_status?: number | null
          created_at?: string
        }
        Relationships: [
          {
            foreignKeyName: "idempotency_keys_tenant_id_fkey"
            columns: ["tenant_id"]
            isOneToOne: false
            referencedRelation: "tenants"
            referencedColumns: ["id"]
          }
        ]
      }
      user_security: {
        Row: {
          id: string
          user_id: string
          email: string
          security_question: string
          security_answer_hash: string
          created_at: string
          updated_at: string
        }
        Insert: {
          id?: string
          user_id: string
          email: string
          security_question: string
          security_answer_hash: string
          created_at?: string
          updated_at?: string
        }
        Update: {
          id?: string
          user_id?: string
          email?: string
          security_question?: string
          security_answer_hash?: string
          created_at?: string
          updated_at?: string
        }
        Relationships: []
      }
    }
    Views: {
      [_ in never]: never
    }
    Functions: {
      current_tenant_ids: {
        Args: Record<PropertyKey, never>
        Returns: string[]
      }
      open_next_period: {
        Args: {
          p_tenant_id: string
          p_label: string
        }
        Returns: string
      }
      cleanup_expired_idempotency_keys: {
        Args: {
          p_ttl_hours?: number
        }
        Returns: number
      }
    }
    Enums: {
      [_ in never]: never
    }
    CompositeTypes: {
      [_ in never]: never
    }
  }
}
