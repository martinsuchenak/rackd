package ipkey_test

import (
	"bytes"
	"context"
	"net/netip"
	"slices"
	"testing"

	"github.com/martinsuchenak/xdal/pkg/migrate"
	"github.com/martinsuchenak/xdal/pkg/xdal"

	"github.com/martinsuchenak/rackd/internal/storage/dialecttest"
	"github.com/martinsuchenak/rackd/pkg/ipkey"
)

func TestRoundTrip(t *testing.T) {
	for _, s := range []string{"0.0.0.0", "10.1.2.3", "255.255.255.255", "::", "::1", "2001:db8::42", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"} {
		key, err := ipkey.Parse(s)
		if err != nil {
			t.Fatal(err)
		}
		if len(key) != ipkey.Size {
			t.Fatalf("%s: key length %d", s, len(key))
		}
		got, err := ipkey.ToAddr(key)
		if err != nil || got.String() != s {
			t.Errorf("%s: round trip gave %v, %v", s, got, err)
		}
	}
	// IPv4-mapped IPv6 and a zone normalise to the plain address.
	a, _ := ipkey.Parse("::ffff:10.0.0.1")
	b, _ := ipkey.Parse("10.0.0.1")
	if !bytes.Equal(a, b) {
		t.Error("::ffff:10.0.0.1 and 10.0.0.1 should share a key")
	}
	z, _ := ipkey.Parse("fe80::1%eth0")
	n, _ := ipkey.Parse("fe80::1")
	if !bytes.Equal(z, n) {
		t.Error("the zone should be dropped")
	}
	if _, err := ipkey.ToAddr([]byte{1, 2, 3}); err != ipkey.ErrInvalidKey {
		t.Errorf("short key: %v", err)
	}
	if k := ipkey.FromAddr(netip.Addr{}); k != nil {
		t.Errorf("zero Addr gave key %x, want nil", k)
	}
}

func TestRange(t *testing.T) {
	for _, c := range []struct{ prefix, first, last string }{
		{"10.1.2.0/24", "10.1.2.0", "10.1.2.255"},
		{"10.1.2.77/24", "10.1.2.0", "10.1.2.255"}, // host bits ignored
		{"192.168.1.10/32", "192.168.1.10", "192.168.1.10"},
		{"0.0.0.0/0", "0.0.0.0", "255.255.255.255"},
		{"10.0.0.0/7", "10.0.0.0", "11.255.255.255"},
		{"2001:db8::/32", "2001:db8::", "2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"},
		{"2001:db8::1/128", "2001:db8::1", "2001:db8::1"},
		{"::/0", "::", "ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff"},
	} {
		start, end, err := ipkey.ParsePrefix(c.prefix)
		if err != nil {
			t.Fatal(err)
		}
		first, _ := ipkey.ToAddr(start)
		last, _ := ipkey.ToAddr(end)
		if first.String() != c.first || last.String() != c.last {
			t.Errorf("%s: %s-%s, want %s-%s", c.prefix, first, last, c.first, c.last)
		}
	}
	if _, _, err := ipkey.Range(netip.Prefix{}); err == nil {
		t.Error("invalid prefix accepted")
	}
}

// Keys sort like the addresses they hold.
func TestKeysSortLikeAddresses(t *testing.T) {
	addrs := []string{"10.0.0.2", "10.0.0.10", "9.255.255.255", "10.0.1.0", "2001:db8::1", "::1", "fe80::1", "0.0.0.0"}
	byKey := slices.Clone(addrs)
	slices.SortFunc(byKey, func(a, b string) int {
		ka, _ := ipkey.Parse(a)
		kb, _ := ipkey.Parse(b)
		return bytes.Compare(ka, kb)
	})
	byAddr := slices.Clone(addrs)
	slices.SortFunc(byAddr, func(a, b string) int {
		return netip.MustParseAddr(a).Unmap().Compare(netip.MustParseAddr(b).Unmap())
	})
	// netip orders IPv4 before IPv6; keys put IPv4 inside ::ffff:0:0/96.
	// Within each family the orders must agree.
	family := func(list []string, v4 bool) []string {
		return slices.DeleteFunc(slices.Clone(list), func(s string) bool { return netip.MustParseAddr(s).Is4() != v4 })
	}
	for _, v4 := range []bool{true, false} {
		if !slices.Equal(family(byKey, v4), family(byAddr, v4)) {
			t.Errorf("v4=%v: key order %v, address order %v", v4, family(byKey, v4), family(byAddr, v4))
		}
	}
}

// The range queries the package documents give the same answers on every
// database backend.
func TestDialectRangeQueries(t *testing.T) {
	steps := []migrate.Migration{{
		Version: "1",
		Name:    "ipkey_test_tables",
		Up: func(ctx context.Context, tx *migrate.Tx) error {
			s := tx.Schema()
			prefixes := s.CreateTable("test_prefixes")
			prefixes.String("cidr", 64).PrimaryKey()
			prefixes.Binary("start_key", ipkey.Size).NotNull()
			prefixes.Binary("end_key", ipkey.Size).NotNull()
			addrs := s.CreateTable("test_addresses")
			addrs.String("ip", 64).PrimaryKey()
			addrs.Binary("ip_key", ipkey.Size).NotNull()
			return tx.Apply(ctx, prefixes, addrs, s.CreateIndex("idx_test_addresses_key", "test_addresses", "ip_key"))
		},
	}}

	dialecttest.Run(t, "ipkey_test", steps, func(t *testing.T, db *dialecttest.DB) {
		ctx := context.Background()
		qb := db.QueryBuilder

		for _, p := range []string{"10.0.0.0/8", "10.1.0.0/16", "10.1.2.0/24", "192.168.0.0/16", "2001:db8::/32", "2001:db8:1::/48"} {
			start, end, _ := ipkey.ParsePrefix(p)
			exec(t, db, qb().Insert("test_prefixes").Set("cidr", p).Set("start_key", start).Set("end_key", end))
		}
		for _, a := range []string{"10.1.2.3", "10.1.2.255", "10.1.3.1", "10.200.0.1", "11.0.0.1", "9.255.255.255", "192.168.5.5", "2001:db8:1::5", "2001:db9::1"} {
			key, _ := ipkey.Parse(a)
			exec(t, db, qb().Insert("test_addresses").Set("ip", a).Set("ip_key", key))
		}

		// Addresses in a prefix, in address order.
		start, end, _ := ipkey.ParsePrefix("10.1.0.0/16")
		got := column(t, db, qb().Select("ip").From("test_addresses").WhereBetween("ip_key", start, end).OrderBy("ip_key ASC"))
		if want := []string{"10.1.2.3", "10.1.2.255", "10.1.3.1"}; !slices.Equal(got, want) {
			t.Errorf("addresses in 10.1.0.0/16: %v, want %v", got, want)
		}

		// Prefixes containing an address, most specific first.
		key, _ := ipkey.Parse("10.1.2.3")
		got = column(t, db, qb().Select("cidr").From("test_prefixes").
			Where("start_key <= ?", key).Where("end_key >= ?", key).OrderBy("start_key DESC"))
		if want := []string{"10.1.2.0/24", "10.1.0.0/16", "10.0.0.0/8"}; !slices.Equal(got, want) {
			t.Errorf("prefixes containing 10.1.2.3: %v, want %v", got, want)
		}

		// Prefixes overlapping a range.
		start, end, _ = ipkey.ParsePrefix("2001:db8:1:2::/64")
		got = column(t, db, qb().Select("cidr").From("test_prefixes").
			Where("start_key <= ?", end).Where("end_key >= ?", start).OrderBy("start_key ASC"))
		if want := []string{"2001:db8::/32", "2001:db8:1::/48"}; !slices.Equal(got, want) {
			t.Errorf("prefixes overlapping 2001:db8:1:2::/64: %v, want %v", got, want)
		}

		// The first used address above one: the step next-free-IP builds on.
		key, _ = ipkey.Parse("10.1.2.3")
		q, args, err := qb().Select("ip_key").From("test_addresses").Where("ip_key > ?", key).OrderBy("ip_key ASC").Limit(1).Build()
		if err != nil {
			t.Fatal(err)
		}
		var next []byte
		if err := db.DB.QueryRow(ctx, q, args...).Scan(&next); err != nil {
			t.Fatal(err)
		}
		if a, _ := ipkey.ToAddr(next); a.String() != "10.1.2.255" {
			t.Errorf("next used address after 10.1.2.3: %v", a)
		}
	})
}

type builder interface {
	Build() (string, []interface{}, error)
}

func exec(t *testing.T, db *dialecttest.DB, b builder) {
	t.Helper()
	q, args, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.DB.Exec(context.Background(), q, args...); err != nil {
		t.Fatalf("%s: %v", q, err)
	}
}

func column(t *testing.T, db *dialecttest.DB, b *xdal.SelectQuery) []string {
	t.Helper()
	q, args, err := b.Build()
	if err != nil {
		t.Fatal(err)
	}
	rows, err := db.DB.Query(context.Background(), q, args...)
	if err != nil {
		t.Fatalf("%s: %v", q, err)
	}
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
