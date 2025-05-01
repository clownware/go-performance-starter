-- 0001_init.up.sql
-- Creates the initial schema with all tables, triggers, and functions

-- First create the updated_at trigger function
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create user_settings table
CREATE TABLE IF NOT EXISTS public.user_settings (
  user_id            uuid PRIMARY KEY REFERENCES auth.users(id),
  stripe_customer_id text,
  created_at         timestamptz NOT NULL DEFAULT now(),
  updated_at         timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_user_settings_updated
  BEFORE UPDATE ON public.user_settings
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Create items table
CREATE TABLE IF NOT EXISTS public.items (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     uuid NOT NULL REFERENCES auth.users(id),
  name        text NOT NULL,
  description text,
  created_at  timestamptz NOT NULL DEFAULT now(),
  updated_at  timestamptz NOT NULL DEFAULT now(),
  CONSTRAINT items_unique_name_per_user UNIQUE (user_id, name)
);

CREATE INDEX IF NOT EXISTS idx_items_user_id ON public.items(user_id);

CREATE TRIGGER trg_items_updated
  BEFORE UPDATE ON public.items
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Create subscriptions table
CREATE TABLE IF NOT EXISTS public.subscriptions (
  user_id               uuid PRIMARY KEY REFERENCES auth.users(id),
  plan_id               text NOT NULL DEFAULT 'free',
  status                text NOT NULL DEFAULT 'active',
  stripe_subscription_id text,
  current_period_end    timestamptz,
  created_at            timestamptz NOT NULL DEFAULT now(),
  updated_at            timestamptz NOT NULL DEFAULT now()
);

CREATE TRIGGER trg_subscriptions_updated
  BEFORE UPDATE ON public.subscriptions
  FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- 0001_init.down.sql
-- Drops all tables, triggers, and functions in reverse order

-- Drop tables (reverse order of creation)
DROP TABLE IF EXISTS public.subscriptions;
DROP TABLE IF EXISTS public.items;
DROP TABLE IF EXISTS public.user_settings;

-- Drop triggers
DROP TRIGGER IF EXISTS trg_subscriptions_updated ON public.subscriptions;
DROP TRIGGER IF EXISTS trg_items_updated ON public.items;
DROP TRIGGER IF EXISTS trg_user_settings_updated ON public.user_settings;

-- Drop function
DROP FUNCTION IF EXISTS set_updated_at();
