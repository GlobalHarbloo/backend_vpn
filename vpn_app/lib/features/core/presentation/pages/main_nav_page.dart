import 'package:flutter/material.dart';
import 'package:vpn_app/features/vpn/presentation/pages/vpn_page.dart';
import 'package:vpn_app/features/profile/presentation/pages/profile_page.dart';
import 'package:vpn_app/features/payments/presentation/pages/payments_page.dart';
import 'package:vpn_app/features/traffic/presentation/pages/traffic_page.dart';

class MainNavPage extends StatefulWidget {
  const MainNavPage({super.key});

  @override
  State<MainNavPage> createState() => _MainNavPageState();
}

class _MainNavPageState extends State<MainNavPage> {
  int _currentIndex = 0;

  final List<Widget> _pages = const [
    VpnPage(),
    TrafficPage(),
    PaymentsPage(),
    ProfilePage(),
  ];

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      body: IndexedStack(index: _currentIndex, children: _pages),
      bottomNavigationBar: BottomNavigationBar(
        type: BottomNavigationBarType.fixed, // Добавлено!
        currentIndex: _currentIndex,
        onTap: (index) => setState(() => _currentIndex = index),
        items: const [
          BottomNavigationBarItem(icon: Icon(Icons.vpn_lock), label: 'VPN'),
          BottomNavigationBarItem(icon: Icon(Icons.bar_chart), label: 'Трафик'),
          BottomNavigationBarItem(icon: Icon(Icons.payment), label: 'Платежи'),
          BottomNavigationBarItem(icon: Icon(Icons.person), label: 'Профиль'),
        ],
      ),
    );
  }
}
